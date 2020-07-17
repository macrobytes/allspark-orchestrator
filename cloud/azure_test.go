package cloud

import (
	"allspark/util/serializer"
	"strconv"
	"testing"
	"time"
)

const (
	azureTemplatePath = "../dist/sample_templates/azure.json"
)

func getVMClient(t *testing.T) CloudEnvironment {
	templateConfig, err := ReadTemplateConfiguration(azureTemplatePath)
	if err != nil {
		t.Fatal(err)
	}

	cloud, err := Create(Azure, templateConfig)
	if err != nil {
		t.Fatal(err)
	}

	return cloud
}

func TestCreateAzureCluster(t *testing.T) {
	cloud := getVMClient(t)
	var spec AzureEnvironment

	err := serializer.DeserializePath(azureTemplatePath, &spec)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cloud.CreateCluster()
	if err != nil {
		t.Fatal(err)
	}

	clusterNodes, err := cloud.getClusterNodes()
	if err != nil {
		t.Error(err)
	}

	expectedNodeCount := spec.WorkerNodes + 1
	actualNodeCount := int64(len(clusterNodes))

	if expectedNodeCount != actualNodeCount {
		t.Error("- expected " + strconv.FormatInt(expectedNodeCount, 10) +
			" spark nodes.")
		t.Error("- got " + strconv.FormatInt(actualNodeCount, 10) +
			" spark nodes.")
	}
}

func TestDestroyAzureCluster(t *testing.T) {
	cloud := getVMClient(t)
	var spec AzureEnvironment

	err := serializer.DeserializePath(azureTemplatePath, &spec)
	if err != nil {
		t.Fatal(err)
	}

	err = cloud.DestroyCluster()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Minute)
	clusterNodes, err := cloud.getClusterNodes()
	if err != nil {
		t.Error(err)
	}

	actualNodeCount := len(clusterNodes)

	if 0 != actualNodeCount {
		t.Error("- expected 0 spark nodes.")
		t.Error("- got " + strconv.Itoa(actualNodeCount) + " spark nodes.")
	}
}