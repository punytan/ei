package ei

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"time"
)

type Client struct {
	Region               string
	AvailabilityZone     string
	ELBName              string
	AutoScalingGroupName string
	Period               int
}

func (c *Client) GetCurrentMetrics() (*cloudwatch.GetMetricStatisticsOutput, *cloudwatch.GetMetricStatisticsOutput, *cloudwatch.GetMetricStatisticsOutput, error) {
	log.Printf("debug: start request")
	rc, err := c.GetRequestCount()
	if err != nil {
		log.Fatal(err)
		return nil, nil, nil, err
	}
	hh, err := c.GetHealthyHostCount()
	if err != nil {
		log.Fatal(err)
		return nil, nil, nil, err
	}
	cu, err := c.GetCPUUtilization()
	if err != nil {
		log.Fatal(err)
		return nil, nil, nil, err
	}

	return rc, hh, cu, nil
}

func (c *Client) GetRequestCount() (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("LoadBalancerName"),
				Value: aws.String(c.ELBName),
			},
			{
				Name:  aws.String("AvailabilityZone"),
				Value: aws.String(c.AvailabilityZone),
			},
		},
		MetricName: aws.String("RequestCount"),
		Namespace:  aws.String("AWS/ELB"),
		Period:     aws.Int64(int64(c.Period)),
		StartTime:  aws.Time(time.Now().Add(-time.Hour * 1)),
		EndTime:    aws.Time(time.Now()),
		Statistics: []*string{aws.String("Sum")},
	}

	return c.get(params)
}

func (c *Client) GetHealthyHostCount() (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("LoadBalancerName"),
				Value: aws.String(c.ELBName),
			},
			{
				Name:  aws.String("AvailabilityZone"),
				Value: aws.String(c.AvailabilityZone),
			},
		},
		MetricName: aws.String("HealthyHostCount"),
		Namespace:  aws.String("AWS/ELB"),
		Period:     aws.Int64(int64(c.Period)),
		StartTime:  aws.Time(time.Now().Add(-time.Hour * 1)),
		EndTime:    aws.Time(time.Now()),
		Statistics: []*string{aws.String("Average")},
	}

	return c.get(params)
}

func (c *Client) GetCPUUtilization() (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("AutoScalingGroupName"),
				Value: aws.String(c.AutoScalingGroupName),
			},
		},
		MetricName: aws.String("CPUUtilization"),
		Namespace:  aws.String("AWS/EC2"),
		Period:     aws.Int64(int64(c.Period)),
		StartTime:  aws.Time(time.Now().Add(-time.Hour * 1)),
		EndTime:    aws.Time(time.Now()),
		Statistics: []*string{aws.String("Average")},
	}

	return c.get(params)
}

func (c *Client) get(params *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	req, resp := cloudwatch.New(sess, aws.NewConfig().WithRegion(c.Region)).GetMetricStatisticsRequest(params)
	if err = req.Send(); err != nil {
		log.Fatal(err)
		return nil, err
	}
	return resp, nil
}
