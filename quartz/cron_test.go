package quartz_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/quartz"
)

func TestCronExpressionAdvanced(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		expected   string
		desc       string
	}{
		{
			expression: "0 */2 * * *", // 每2小时整点
			expected:   "Mon Jan 1 14:00:00 2024",
			desc:       "every 2 hours",
		},
		{
			expression: "45 9 */5 * *", // 每5天的9:45
			expected:   "Sat Jan 6 09:45:00 2024",
			desc:       "every 5 days at 9:45",
		},
		{
			expression: "0 12 * * 1", // 每周一12点（更清晰）
			expected:   "Mon Jan 8 12:00:00 2024",
			desc:       "every Monday at noon",
		},
		{
			expression: "15,45 * * * 0,6", // 周末每小时的15分和45分
			expected:   "Sat Jan 6 00:15:00 2024",
			desc:       "weekends at 15 and 45 minutes",
		},
		{
			expression: "0 */2 * * 1-5", // 工作日每2小时（更简单）
			expected:   "Mon Jan 1 14:00:00 2024",
			desc:       "weekdays every 2 hours",
		},
		{
			expression: "30 14 15 */3 *", // 每3个月15号14:30
			expected:   "Mon Jan 15 14:30:00 2024",
			desc:       "15th day every 3 months",
		},
		{
			expression: "0 0 29-31 2 *",            // 2月29-31号（闰年测试）
			expected:   "Thu Feb 29 00:00:00 2024", // 2024是闰年
			desc:       "leap year February 29-31",
		},
		{
			expression: "5 10 * 1,7 *", // 1月和7月每天10:05
			expected:   "Tue Jan 2 10:05:00 2024",
			desc:       "January and July daily",
		},
		{
			expression: "0 6,18 * * 1,3,5", // 周一三五的6点和18点
			expected:   "Mon Jan 1 18:00:00 2024",
			desc:       "Mon/Wed/Fri at 6 and 18",
		},
		{
			expression: "*/10 * * * *", // 每10分钟
			expected:   "Mon Jan 1 12:10:00 2024",
			desc:       "every 10 minutes",
		},
	}

	prev := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	for _, tt := range tests {
		test := tt
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, time.UTC, 1)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		startTime  time.Time
		expected   string
		desc       string
	}{
		{
			expression: "59 23 31 12 *", // 12月31号23:59
			startTime:  time.Date(2024, 12, 31, 20, 0, 0, 0, time.UTC),
			expected:   "Tue Dec 31 23:59:00 2024",
			desc:       "end of year",
		},
		{
			expression: "0 0 1 1 *", // 新年第一秒
			startTime:  time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			expected:   "Wed Jan 1 00:00:00 2025",
			desc:       "new year",
		},
		{
			expression: "30 2 29 2 *", // 闰年2月29号2:30
			startTime:  time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC),
			expected:   "Thu Feb 29 02:30:00 2024",
			desc:       "leap day",
		},
		{
			expression: "0 12 30 4,6,9,11 *",
			startTime:  time.Date(2024, 4, 29, 0, 0, 0, 0, time.UTC),
			expected:   "Tue Apr 30 12:00:00 2024",
			desc:       "last day of 30-day months",
		},
		{
			expression: "0 12 31 1,3,5,7,8,10,12 *",
			startTime:  time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC),
			expected:   "Fri May 31 12:00:00 2024",
			desc:       "31st day of 31-day months",
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(test.startTime.UnixMilli(), cronTrigger, time.UTC, 1)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionBoundaryValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		expected   string
		desc       string
	}{
		{
			expression: "0 0 * * *", // 每天午夜
			expected:   "Tue Jan 2 00:00:00 2024",
			desc:       "daily at midnight",
		},
		{
			expression: "59 23 * * *", // 每天23:59
			expected:   "Mon Jan 1 23:59:00 2024",
			desc:       "daily at 23:59",
		},
		{
			expression: "0 0 1 * *", // 每月1号
			expected:   "Thu Feb 1 00:00:00 2024",
			desc:       "monthly on 1st",
		},
		{
			expression: "0 0 * 1 *", // 每年1月每天
			expected:   "Tue Jan 2 00:00:00 2024",
			desc:       "daily in January",
		},
		{
			expression: "0 0 * 12 *", // 每年12月每天
			expected:   "Sun Dec 1 00:00:00 2024",
			desc:       "daily in December",
		},
	}

	prev := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	for _, tt := range tests {
		test := tt
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, time.UTC, 1)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionMultipleIterations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		iterations int
		expected   string
		desc       string
	}{
		{
			expression: "*/5 * * * *", // 每5分钟
			iterations: 12,            // 1小时后
			expected:   "Mon Jan 1 13:00:00 2024",
			desc:       "every 5 minutes - 12 iterations",
		},
		{
			expression: "0 */4 * * *", // 每4小时
			iterations: 6,             // 24小时后
			expected:   "Tue Jan 2 12:00:00 2024",
			desc:       "every 4 hours - 6 iterations",
		},
		{
			expression: "0 9 * * 1-5", // 工作日9点
			iterations: 5,             // 5个工作日
			expected:   "Mon Jan 8 09:00:00 2024",
			desc:       "weekdays at 9 - 5 iterations",
		},
	}

	prev := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	for _, tt := range tests {
		test := tt
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, time.UTC, test.iterations)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionBasic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		startTime  time.Time
		expected   string
	}{
		{
			name:       "every_15_minutes",
			expression: "*/15 * * * *",
			startTime:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			expected:   "Mon Jan 1 12:15:00 2024",
		},
		{
			name:       "daily_at_noon",
			expression: "0 12 * * *",
			startTime:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			expected:   "Mon Jan 1 12:00:00 2024",
		},
		{
			name:       "weekdays_at_9am",
			expression: "0 9 * * 1-5",
			startTime:  time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC), // Monday
			expected:   "Mon Jan 1 09:00:00 2024",
		},
		{
			name:       "first_of_month",
			expression: "0 0 1 * *",
			startTime:  time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			expected:   "Thu Feb 1 00:00:00 2024",
		},
		{
			name:       "weekend_mornings",
			expression: "0 8 * * 0,6",
			startTime:  time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC), // Friday
			expected:   "Sat Jan 6 08:00:00 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cronTrigger, err := quartz.NewCronTrigger(tt.expression)
			assert.IsNil(t, err)

			nextTime, err := cronTrigger.NextFireTime(tt.startTime.UnixMilli())
			assert.IsNil(t, err)

			result := formatTime(nextTime, time.UTC)
			assert.Equal(t, result, tt.expected)
		})
	}
}

func TestCronExpressionComplexPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		startTime  time.Time
		expected   string
	}{
		{
			name:       "every_2_hours_workdays",
			expression: "0 */2 * * 1-5",
			startTime:  time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), // Monday 13:00
			expected:   "Mon Jan 1 14:00:00 2024",
		},
		{
			name:       "multiple_minutes",
			expression: "15,45 10,14 * * *",
			startTime:  time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			expected:   "Mon Jan 1 10:15:00 2024",
		},
		{
			name:       "specific_days_range",
			expression: "30 16 5-10 * *",
			startTime:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			expected:   "Fri Jan 5 16:30:00 2024",
		},
		{
			name:       "quarterly_first_monday",
			expression: "0 9 1-7 1,4,7,10 1",
			startTime:  time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC), // Jan 1 is Monday
			expected:   "Mon Jan 1 09:00:00 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cronTrigger, err := quartz.NewCronTrigger(tt.expression)
			assert.IsNil(t, err)

			nextTime, err := cronTrigger.NextFireTime(tt.startTime.UnixMilli())
			assert.IsNil(t, err)

			result := formatTime(nextTime, time.UTC)
			assert.Equal(t, result, tt.expected)
		})
	}
}

func iterate(prev int64, cronTrigger *quartz.CronTrigger, loc *time.Location,
	iterations int) (string, error) {
	var err error
	for i := 0; i < iterations; i++ {
		prev, err = cronTrigger.NextFireTime(prev)
		// log.Print(formatTime(prev, loc))
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	}
	return formatTime(prev, loc), nil
}

const readDateLayout = "Mon Jan 2 15:04:05 2006"

func formatTime(t int64, loc *time.Location) string {
	return time.UnixMilli(t).In(loc).Format(readDateLayout)
}
