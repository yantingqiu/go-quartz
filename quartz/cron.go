package quartz

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type CronTrigger struct {
	expression string
	schedule   cron.Schedule
	location   *time.Location
}

var _ Trigger = (*CronTrigger)(nil)

// NewCronTrigger returns a new [CronTrigger] using the UTC location.
func NewCronTrigger(expression string) (*CronTrigger, error) {
	return NewCronTriggerWithLoc(expression, time.UTC)
}

// NewCronTriggerWithLoc returns a new [CronTrigger] with the given [time.Location].
func NewCronTriggerWithLoc(expression string, location *time.Location) (*CronTrigger, error) {
	if location == nil {
		return nil, newIllegalArgumentError("location is nil")
	}

	if expression == "" {
		return nil, newIllegalArgumentError("cron expression cannot be empty")
	}

	parser := cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	schedule, err := parser.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron expression '%s': %w", expression, err)
	}

	return &CronTrigger{
		expression: expression,
		location:   location,
		schedule:   schedule,
	}, nil
}

func (ct *CronTrigger) NextFireTime(prev int64) (int64, error) {
	var baseTime time.Time

	if prev <= 0 {
		baseTime = time.Now().In(ct.location)
	} else {
		baseTime = time.UnixMilli(prev).In(ct.location)
	}

	nextTime := ct.schedule.Next(baseTime)

	if nextTime.IsZero() {
		return -1, fmt.Errorf("no next fire time available for cron expression: %s", ct.expression)
	}

	return nextTime.UnixMilli(), nil
}

func (ct *CronTrigger) Description() string {
	return fmt.Sprintf("CronTrigger%s%s%s%s", Sep, ct.expression, Sep, ct.location.String())
}

func (ct *CronTrigger) GetExpression() string {
	return ct.expression
}

func (ct *CronTrigger) GetLocation() *time.Location {
	return ct.location
}
