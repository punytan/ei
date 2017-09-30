package ei

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"math"
	"sort"
	"time"
)

type Point struct {
	Timestamp           time.Time
	ELBRequestCount     float64
	ELBHealthyHostCount float64
	AutoScalingGroupCPU float64
}

func NewPoint(r *cloudwatch.Datapoint) *Point {
	p := new(Point)
	p.Timestamp = *r.Timestamp
	return p
}

func (p Point) CPUUtilizationPerRequest() float64 {
	return p.ELBRequestCount / p.ELBHealthyHostCount / p.AutoScalingGroupCPU
}

func (p Point) RequestCountPerHost() float64 {
	return p.ELBRequestCount / p.ELBHealthyHostCount
}

func (p Point) RequiredHostCount(ratio float64) float64 {
	return p.ELBRequestCount / p.CPUUtilizationPerRequest() / 100 / ratio
}

func (p Point) EstimatedRequiredHostCount(cpuPerReq float64, ratio float64) float64 {
	return p.ELBRequestCount / cpuPerReq / 100 / ratio
}

func (p Point) EstimatedCurrentCPUUtilization(recentAverageCPUReq float64) float64 {
	return p.ELBRequestCount / recentAverageCPUReq / p.ELBHealthyHostCount
}

type PointContainer struct {
	Points map[string]*Point
}

func NewPointContainer(rc *cloudwatch.GetMetricStatisticsOutput, hh *cloudwatch.GetMetricStatisticsOutput, cu *cloudwatch.GetMetricStatisticsOutput) *PointContainer {
	points := map[string]*Point{}
	for i := range rc.Datapoints {
		key := rc.Datapoints[i].Timestamp.String()
		if _, ok := points[key]; !ok {
			points[key] = NewPoint(rc.Datapoints[i])
		}
		points[key].ELBRequestCount = *rc.Datapoints[i].Sum
	}

	for i := range hh.Datapoints {
		key := hh.Datapoints[i].Timestamp.String()
		if _, ok := points[key]; !ok {
			points[key] = NewPoint(hh.Datapoints[i])
		}
		points[key].ELBHealthyHostCount = *hh.Datapoints[i].Average
	}

	for i := range cu.Datapoints {
		key := cu.Datapoints[i].Timestamp.String()
		if _, ok := points[key]; !ok {
			points[key] = NewPoint(cu.Datapoints[i])
		}
		points[key].AutoScalingGroupCPU = *cu.Datapoints[i].Average
	}

	pc := &PointContainer{
		Points: points,
	}

	keys := pc.Keys()
	delete(pc.Points, keys[len(keys)-1]) // dismiss ambiguous datapoint

	return pc
}

func (pc PointContainer) Keys() []string {
	sortedTimes := make([]time.Time, 0, len(pc.Points))

	for _, value := range pc.Points {
		sortedTimes = append(sortedTimes, value.Timestamp)
	}

	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i].Before(sortedTimes[j])
	})

	sortedKeys := make([]string, 0, len(sortedTimes))
	for i := range sortedTimes {
		sortedKeys = append(sortedKeys, sortedTimes[i].String())
	}

	return sortedKeys
}

func (pc PointContainer) RecentAverageCPUReq() float64 {
	keys := pc.Keys()
	count := 0
	total := 0.0
	for i := len(keys) - 1; i >= 0; i-- {
		if math.IsInf(pc.Points[keys[i]].CPUUtilizationPerRequest(), 0) {
			continue
		}

		if count > 5 {
			break
		}

		total += pc.Points[keys[i]].CPUUtilizationPerRequest()
		count++
	}

	return total / float64(count)
}

func (pc PointContainer) GetLatestPoint() *Point {
	keys := pc.Keys()
	return pc.Points[keys[len(keys)-1]]
}
