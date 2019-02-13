package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type singleRegionAWSManager struct {
	region    string
	svc       *ec2.EC2
	instances []Instance
}

// RnDTag is a filter for slave instances
const RnDTag = "R&D"

// RnDMasterTag is a filter for master instance
const RnDMasterTag = "R&D_master"

const running = "running"

//NewAWS creates AWS manager for single region
func NewAWS(region string) Manager {
	sess := awsSession(region)
	// Create EC2 service client
	svc := ec2.New(sess)

	awsM := &singleRegionAWSManager{
		region: region,
		svc:    svc,
	}

	_, err := awsM.RefreshInstances()
	if err != nil {
		panic(err)
	}
	return awsM
}

func awsSession(region string) *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		panic(err)
	}
	return sess
}

//refreshes list of aws instances
func (a *singleRegionAWSManager) RefreshInstances() ([]Instance, error) {
	var awsTags []*string
	for _, tag := range []string{RnDTag, RnDMasterTag} {
		awsTags = append(awsTags, aws.String(tag))
	}

	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("tag:Name"),
				Values: awsTags,
			},
		},
	}
	result, err := a.svc.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	var instances []Instance
	for _, reservation := range result.Reservations {
		for _, i := range reservation.Instances {
			id := i.InstanceId
			state := i.State.Name
			if *state == "terminated" {
				continue
			}
			pubIP := i.PublicIpAddress
			for _, tag := range i.Tags {
				if *tag.Value == RnDMasterTag {
					inst := Instance{id, pubIP, state, a.region, *tag.Value, nil}
					instances = append(instances, inst)
				}
				if *tag.Value == RnDTag {
					inst := Instance{id, pubIP, state, a.region, *tag.Value, nil}
					instances = append(instances, inst)
				}
			}
		}
	}
	a.instances = instances
	return instances, nil
}

func (a *singleRegionAWSManager) Instances() []Instance {
	return a.instances
}

func (a *singleRegionAWSManager) StartInstances() error {

	if len(a.Instances()) == 0 {
		return nil
	}
	// We set DryRun to true to check to see if the instance exists and we have the
	// necessary permissions to monitor the instance.
	instanceIds := instanceToInstanceID(a.instances)
	input := &ec2.StartInstancesInput{
		InstanceIds: instanceIds,
		DryRun:      aws.Bool(true),
	}

	_, err := a.svc.StartInstances(input)
	awsErr, ok := err.(awserr.Error)

	// If the error code is `DryRunOperation` it means we have the necessary
	// permissions to Start this instance
	if ok && awsErr.Code() == "DryRunOperation" {
		// Let's now set dry run to be false. This will allow us to start the instances
		input.DryRun = aws.Bool(false)
		_, err = a.svc.StartInstances(input)
		if err != nil {
			return err
		}
		_, err := WaitUntilAllInstancesRunning(a, func() {
			fmt.Println("Waiting for amazon instances to start")
			time.Sleep(20 * time.Second)
		})
		if err != nil {
			return err
		}
		return nil
	}
	// This could be due to a lack of permissions
	return err
}

func (a *singleRegionAWSManager) StopInstances() error {
	instanceIds := instanceToInstanceID(a.instances)
	input := &ec2.StopInstancesInput{
		InstanceIds: instanceIds,
		DryRun:      aws.Bool(true),
	}

	_, err := a.svc.StopInstances(input)
	awsErr, ok := err.(awserr.Error)
	if ok && awsErr.Code() == "DryRunOperation" {
		input.DryRun = aws.Bool(false)
		_, err = a.svc.StopInstances(input)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}
