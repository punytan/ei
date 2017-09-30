package ei

import (
	"flag"
	"fmt"
	"io"
	"log"
	"time"
)

const (
	ExitCodeOK = iota
	ExitCodeParserFlagError
	ExitCodeFatal
)

type CLI struct {
	OutStream, ErrStream io.Writer
	Version              string
	Name                 string
}

type Option struct {
	version              bool
	region               string
	elbName              string
	availabilityZone     string
	autoscalingGroupName string
	period               int
	upperCPUThreshold    float64
	middleCPUThreshold   float64
	lowerCPUThreshold    float64
}

func (c *CLI) ParseArgs(args []string) *Option {
	option := &Option{}

	flags := flag.NewFlagSet(c.Name, flag.ContinueOnError)
	flags.SetOutput(c.ErrStream)
	flags.BoolVar(&option.version, "version", false, "print version infomation and quit")
	flags.StringVar(&option.region, "region", "", "AWS region")
	flags.StringVar(&option.elbName, "elb-name", "", "target ELB name")
	flags.StringVar(&option.availabilityZone, "availability-zone", "", "target availability zone")
	flags.StringVar(&option.autoscalingGroupName, "autoscaling-group-name", "", "target AutoScaling group name")
	flags.IntVar(&option.period, "period", 60, "target period")
	flags.Float64Var(&option.upperCPUThreshold, "upper-cpu-threshold", 0.60, "CPU upper threshold")
	flags.Float64Var(&option.lowerCPUThreshold, "lower-cpu-threshold", 0.40, "CPU lower threshold")

	if err := flags.Parse(args[1:]); err != nil {
		return nil
	}

	if option == nil {
		return nil
	}

	if option.version {
		fmt.Fprintf(c.ErrStream, "%s version %s\n", c.Name, c.Version)
		return nil
	}

	if option.region == "" {
		log.Printf("error: specify `--region` option")
		return nil
	}

	if option.elbName == "" {
		log.Printf("error: specify `--elb-name` option")
		return nil
	}

	if option.availabilityZone == "" {
		log.Printf("error: specify `--availability-zone` option")
		return nil
	}

	if option.autoscalingGroupName == "" {
		log.Printf("error: specify `--autoscaling-group-name` option")
		return nil
	}

	if option.period < 15 {
		log.Printf("error: specify `--period` option greater than 15")
		return nil
	}

	option.middleCPUThreshold = (option.upperCPUThreshold + option.lowerCPUThreshold) / 2
	return option
}

func (c *CLI) Run(args []string) int {
	option := c.ParseArgs(args)
	if option == nil {
		return ExitCodeParserFlagError
	}

	fmt.Fprintf(c.OutStream, "%v\n", option)

	client := Client{
		Region:               option.region,
		AvailabilityZone:     option.availabilityZone,
		ELBName:              option.elbName,
		AutoScalingGroupName: option.autoscalingGroupName,
		Period:               option.period,
	}

	rc, hh, cu, err := client.GetCurrentMetrics()
	if err != nil {
		log.Fatal(err)
		return ExitCodeFatal
	}

	pc := NewPointContainer(rc, hh, cu)

	sortedKeys := pc.Keys()
	for i := range sortedKeys {
		k := sortedKeys[i]
		v := pc.Points[k]
		log.Printf("info: [%s] ReqCount: %.0f, HostCount: %.0f, CPU: %6.2f(%6.2f), ReqPerHost: %.2f, Required: %3.0f, %3.0f, %3.0f; E-Host %.2f; E-CPURAvg: %.2f, RecentAgv: %.2f",
			v.Timestamp.In(time.FixedZone("Asia/Tokyo", 9*60*60)),
			v.ELBRequestCount,
			v.ELBHealthyHostCount,
			v.AutoScalingGroupCPU,
			v.CPUUtilizationPerRequest(),
			v.RequestCountPerHost(),
			v.RequiredHostCount(option.upperCPUThreshold),
			v.RequiredHostCount(option.middleCPUThreshold),
			v.RequiredHostCount(option.lowerCPUThreshold),
			v.EstimatedRequiredHostCount(pc.RecentAverageCPUReq(), option.middleCPUThreshold),
			v.EstimatedCurrentCPUUtilization(pc.RecentAverageCPUReq()),
			pc.RecentAverageCPUReq(),
		)
	}

	p := pc.GetLatestPoint()
	log.Printf("info: EstimatedCurrentCPUUtilization: %.2f", p.EstimatedCurrentCPUUtilization(pc.RecentAverageCPUReq()))
	log.Printf("info: EstimatedRequiredHostCount(%.0f%%) %.0f", option.upperCPUThreshold*100, p.EstimatedRequiredHostCount(pc.RecentAverageCPUReq(), option.upperCPUThreshold))
	log.Printf("info: EstimatedRequiredHostCount(%.0f%%) %.0f", option.middleCPUThreshold*100, p.EstimatedRequiredHostCount(pc.RecentAverageCPUReq(), option.middleCPUThreshold))
	log.Printf("info: EstimatedRequiredHostCount(%.0f%%) %.0f", option.lowerCPUThreshold*100, p.EstimatedRequiredHostCount(pc.RecentAverageCPUReq(), option.lowerCPUThreshold))

	return ExitCodeOK
}
