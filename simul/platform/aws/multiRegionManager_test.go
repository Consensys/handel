package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"
)

const stopped = "stopped"

type mockSingleRegionManager struct {
	instances []Instance
	region    string
}

func (a *mockSingleRegionManager) Instances() []Instance {
	return a.instances
}

func (a *mockSingleRegionManager) StartInstances() error {
	for _, inst := range a.Instances() {
		*inst.State = running
	}
	return nil
}

func (a *mockSingleRegionManager) StopInstances() error {
	for _, inst := range a.Instances() {
		*inst.State = stopped
	}
	return nil
}

func (a *mockSingleRegionManager) RefreshInstances() ([]Instance, error) {
	return a.instances, nil
}

func newMultiRegionAWSManager() Manager {
	return &multiRegionAWSManager{}
}

func makeManager(n int, reg string) Manager {
	var instances []Instance
	for i := 0; i < n; i++ {
		inst := Instance{
			ID:     aws.String(string(n) + reg),
			State:  aws.String(stopped),
			region: reg,
		}
		instances = append(instances, inst)
	}
	return &mockSingleRegionManager{
		instances: instances,
		region:    reg,
	}
}

func newMockManager() (Manager, map[string]int) {
	regMap := make(map[string]int)
	reg1 := "us-east-1"
	regMap[reg1] = 3
	reg2 := "ap-south-1"
	regMap[reg2] = 5
	reg3 := "cn-north-1"
	regMap[reg3] = 0

	m1 := makeManager(regMap[reg1], reg1)
	m2 := makeManager(regMap[reg2], reg2)
	m3 := makeManager(regMap[reg3], reg3)

	manager := multiRegionAWSManager{[]Manager{m1, m2, m3}}
	return &manager, regMap
}

func TestMultiRegionManager(t *testing.T) {
	manager, regMap := newMockManager()
	freshInstances, _ := manager.RefreshInstances()
	for _, inst := range freshInstances {
		require.Equal(t, *inst.State, stopped)
	}

	manager.StopInstances()
	freshInstances, _ = manager.RefreshInstances()
	for _, inst := range freshInstances {
		require.Equal(t, *inst.State, stopped)
	}

	manager.StartInstances()
	freshInstances, _ = manager.RefreshInstances()
	for _, inst := range freshInstances {
		require.Equal(t, *inst.State, running)
	}

	manager.StartInstances()
	freshInstances, _ = manager.RefreshInstances()
	for _, inst := range freshInstances {
		require.Equal(t, *inst.State, running)
	}

	manager.StopInstances()
	freshInstances, _ = manager.RefreshInstances()
	for _, inst := range freshInstances {
		require.Equal(t, *inst.State, stopped)
	}

	for _, inst := range manager.Instances() {
		regMap[inst.Region] = regMap[inst.Region] - 1
	}

	for _, v := range regMap {
		require.Equal(t, v, 0)
	}
}

func TestAllInstancesRunningBlock(t *testing.T) {
	manager, _ := newMockManager()
	k := 0
	tries := k
	delay := func() {
		tries = tries + 1
		if tries >= 3 {
			manager.StartInstances()
		}
	}
	attempts, err := WaitUntilAllInstancesRunning(manager, delay)
	require.Nil(t, err)
	require.Equal(t, attempts, tries)

	insances := manager.Instances()
	for _, inst := range insances {
		require.Equal(t, *inst.State, running)
	}
}
