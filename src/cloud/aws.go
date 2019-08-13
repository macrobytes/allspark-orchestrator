package cloud

import (
	b64 "encoding/base64"
	"log"
	"strconv"
	"template"
	"time"
	"util/netutil"
	"util/template_reader"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type AwsEnvironment struct {
	region string
}

func (e *AwsEnvironment) getEc2Client() *ec2.EC2 {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(e.region)},
	)

	if err != nil {
		log.Fatal(err)
	}

	return ec2.New(sess)
}

func (e *AwsEnvironment) launchInstances(template template.AwsTemplate,
	identifier string, instanceCount int64, userData string) (*ec2.Reservation, error) {

	cli := e.getEc2Client()

	encodedUserData := b64.StdEncoding.EncodeToString([]byte(userData))

	resp, err := cli.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String(template.ImageId),
		InstanceType:     aws.String(template.InstanceType),
		MinCount:         aws.Int64(instanceCount),
		MaxCount:         aws.Int64(instanceCount),
		SecurityGroupIds: aws.StringSlice(template.SecurityGroupIds),
		SubnetId:         aws.String(template.SubnetId),
		UserData:         aws.String(encodedUserData),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String(template.IAMRole),
		},

		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					Encrypted:  aws.Bool(true),
					VolumeSize: aws.Int64(template.EBSVolumeSize),
					VolumeType: aws.String("gp2"),
				},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for _, el := range resp.Instances {
		_, err := cli.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{el.InstanceId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(identifier),
				},
			},
		})

		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}

func (e *AwsEnvironment) getPublicIp(instanceId string) (string, error) {
	cli := e.getEc2Client()

	cli.WaitUntilInstanceRunning(
		&ec2.DescribeInstancesInput{
			InstanceIds: aws.StringSlice([]string{instanceId}),
		},
	)

	response, err := cli.DescribeInstances(
		&ec2.DescribeInstancesInput{
			InstanceIds: aws.StringSlice([]string{instanceId}),
		},
	)

	if err != nil {
		return "", err
	}

	return *response.Reservations[0].Instances[0].PublicIpAddress, nil
}

func (e *AwsEnvironment) launchMaster(template template.AwsTemplate,
	baseIdentifier string) (string, string, error) {

	workers := strconv.FormatInt(template.WorkerNodes, 10)
	userData := "export EXPECTED_WORKERS=" + workers

	res, err := e.launchInstances(template, baseIdentifier+MASTER_IDENTIFIER,
		1, userData)
	if err != nil {
		return "", "", err
	}

	privateIp := *res.Instances[0].PrivateIpAddress

	return *res.Instances[0].InstanceId, privateIp, err
}

func (e *AwsEnvironment) launchWorkers(template template.AwsTemplate,
	baseIdentifier string, masterIP string) (*ec2.Reservation, error) {

	userData := "export MASTER_IP=" + masterIP

	return e.launchInstances(template,
		baseIdentifier+WORKER_IDENTIFIER,
		template.WorkerNodes,
		userData)
}

func (e *AwsEnvironment) CreateCluster(templatePath string) (string, error) {
	var awsTemplate template.AwsTemplate
	err := template_reader.Deserialize(templatePath, &awsTemplate)
	if err != nil {
		log.Fatal(err)
	}
	e.region = awsTemplate.Region

	baseIdentifier := buildBaseIdentifier(awsTemplate.ClusterID)
	instanceId, privateIp, err := e.launchMaster(awsTemplate, baseIdentifier)
	if err != nil {
		return "", err
	}
	_, err = e.launchWorkers(awsTemplate, baseIdentifier, privateIp)

	publicIp, err := e.getPublicIp(instanceId)
	if err != nil {
		return "", err
	}

	if netutil.IsListeningOnPort(publicIp, 8080, 1*time.Second, 60) {
		log.Println("spark master node is online")
	}

	webUrl := "http://" + publicIp + ":8080"
	return webUrl, err
}

func (e *AwsEnvironment) DestroyCluster(templatePath string) error {
	var awsTemplate template.AwsTemplate
	err := template_reader.Deserialize(templatePath, &awsTemplate)
	if err != nil {
		log.Fatal(err)
	}
	e.region = awsTemplate.Region

	identifier := awsTemplate.ClusterID

	cli := e.getEc2Client()
	instances, err := e.getClusterNodes(identifier)
	if err != nil {
		return err
	}

	_, err = cli.TerminateInstances(
		&ec2.TerminateInstancesInput{
			InstanceIds: aws.StringSlice(instances),
		},
	)
	return err
}

func (e *AwsEnvironment) getClusterNodes(identifier string) ([]string, error) {
	var instances []string

	cli := e.getEc2Client()
	resp, err := cli.DescribeInstances(
		&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("tag:Name"),
					Values: aws.StringSlice([]string{identifier + "*"}),
				},

				{
					Name:   aws.String("instance-state-name"),
					Values: aws.StringSlice([]string{"running", "pending"}),
				},
			},
		},
	)

	if err != nil {
		return instances, err
	}

	for _, reservation := range resp.Reservations {
		for _, el := range reservation.Instances {
			instances = append(instances, *el.InstanceId)
		}
	}

	return instances, nil
}
