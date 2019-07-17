package cloud

import (
	"strconv"
	"template"
	"testing"
	"util/template_reader"
)

func TestCreateAwsCluster(t *testing.T) {
	var template template.AwsTemplate
	template_reader.Deserialize("../../sample_templates/aws.json",
		&template)

	cloud := Create(AWS)
	err := cloud.CreateCluster("../../sample_templates/aws.json")
	if err != nil {
		t.Fatal(err)
	}

	clusterNodes, err := cloud.getClusterNodes(template.ClusterID)
	if err != nil {
		t.Error(err)
	}

	expectedNodeCount := template.WorkerNodes + 1
	actualNodeCount := int64(len(clusterNodes))

	if expectedNodeCount != actualNodeCount {
		t.Error("- expected " + strconv.FormatInt(expectedNodeCount, 10) +
			" spark nodes.")
		t.Error("- got " + strconv.FormatInt(actualNodeCount, 10) +
			" spark nodes.")
	}
}

func TestDestroyAwsCluster(t *testing.T) {
	var template template.AwsTemplate
	template_reader.Deserialize("../../sample_templates/aws.json",
		&template)

	cloud := Create(AWS)
	cloud.DestroyCluster(template.ClusterID)

	clusterNodes, err := cloud.getClusterNodes(template.ClusterID)
	if err != nil {
		t.Error(err)
	}

	actualNodeCount := len(clusterNodes)

	if 0 != actualNodeCount {
		t.Error("- expected 0 spark nodes.")
		t.Error("- got " + strconv.Itoa(actualNodeCount) + " spark nodes.")
	}
}
