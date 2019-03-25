// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxtime encapsulates some golang.time functions
// refer to github.com/jinzhu/now
package gxtime

import (
	"errors"
	"regexp"
	"time"
)

//
//  now.BeginningOfMinute() // 2013-11-18 17:51:00 Mon
//  now.BeginningOfDay()    // 2013-11-18 00:00:00 Mon
//  now.EndOfDay()          // 2013-11-18 23:59:59.999999999 Mon

var FirstDayMonday bool
var TimeFormats = []string{"1/2/2006", "1/2/2006 15:4:5", "2006-1-2 15:4:5", "2006-1-2 15:4", "2006-1-2", "1-2", "15:4:5", "15:4", "15", "15:4:5 Jan 2, 2006 MST", "2006-01-02 15:04:05.999999999 -0700 MST"}

// parse time string
type GxTime struct {
	time.Time
}

func NewGxTime(t time.Time) *GxTime {
	return &GxTime{t}
}

func BeginningOfMinute() time.Time {
	return NewGxTime(time.Now()).BeginningOfMinute()
}

func BeginningOfHour() time.Time {
	return NewGxTime(time.Now()).BeginningOfHour()
}

func BeginningOfDay() time.Time {
	return NewGxTime(time.Now()).BeginningOfDay()
}

func BeginningOfWeek() time.Time {
	return NewGxTime(time.Now()).BeginningOfWeek()
}

func BeginningOfMonth() time.Time {
	return NewGxTime(time.Now()).BeginningOfMonth()
}

func BeginningOfQuarter() time.Time {
	return NewGxTime(time.Now()).BeginningOfQuarter()
}

func BeginningOfYear() time.Time {
	return NewGxTime(time.Now()).BeginningOfYear()
}

func EndOfMinute() time.Time {
	return NewGxTime(time.Now()).EndOfMinute()
}

func EndOfHour() time.Time {
	return NewGxTime(time.Now()).EndOfHour()
}

func EndOfDay() time.Time {
	return NewGxTime(time.Now()).EndOfDay()
}

func EndOfWeek() time.Time {
	return NewGxTime(time.Now()).EndOfWeek()
}

func EndOfMonth() time.Time {
	return NewGxTime(time.Now()).EndOfMonth()
}

func EndOfQuarter() time.Time {
	return NewGxTime(time.Now()).EndOfQuarter()
}

func EndOfYear() time.Time {
	return NewGxTime(time.Now()).EndOfYear()
}

func Monday() time.Time {
	return NewGxTime(time.Now()).Monday()
}

func Sunday() time.Time {
	return NewGxTime(time.Now()).Sunday()
}

func EndOfSunday() time.Time {
	return NewGxTime(time.Now()).EndOfSunday()
}

func Parse(strs ...string) (time.Time, error) {
	return NewGxTime(time.Now()).Parse(strs...)
}

func ParseInLocation(loc *time.Location, strs ...string) (time.Time, error) {
	return NewGxTime(time.Now().In(loc)).Parse(strs...)
}

func MustParse(strs ...string) time.Time {
	return NewGxTime(time.Now()).MustParse(strs...)
}

func MustParseInLocation(loc *time.Location, strs ...string) time.Time {
	return NewGxTime(time.Now().In(loc)).MustParse(strs...)
}

func Between(time1, time2 string) bool {
	return NewGxTime(time.Now()).Between(time1, time2)
}

func (now *GxTime) BeginningOfMinute() time.Time {
	return now.Truncate(time.Minute)
}

func (now *GxTime) BeginningOfHour() time.Time {
	return now.Truncate(time.Hour)
}

func (now *GxTime) BeginningOfDay() time.Time {
	//d := time.Duration(-now.Hour()) * time.Hour
	//return now.BeginningOfHour().Add(d)
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

func (now *GxTime) BeginningOfWeek() time.Time {
	t := now.BeginningOfDay()
	weekday := int(t.Weekday())
	if FirstDayMonday {
		if weekday == 0 {
			weekday = 7
		}
		weekday = weekday - 1
	}

	//d := time.Duration(-weekday) * 24 * time.Hour
	//return t.Add(d)
	return t.AddDate(0, 0, -weekday)
}

func (now *GxTime) BeginningOfMonth() time.Time {
	//t := now.BeginningOfDay()
	//d := time.Duration(-int(t.Day())+1) * 24 * time.Hour
	//return t.Add(d)
	y, m, _ := now.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, now.Location())
}

func (now *GxTime) BeginningOfQuarter() time.Time {
	month := now.BeginningOfMonth()
	offset := (int(month.Month()) - 1) % 3
	return month.AddDate(0, -offset, 0)
}

func (now *GxTime) BeginningOfYear() time.Time {
	//t := now.BeginningOfDay()
	//d := time.Duration(-int(t.YearDay())+1) * 24 * time.Hour
	//return t.Truncate(time.Hour).Add(d)
	y, _, _ := now.Date()
	return time.Date(y, time.January, 1, 0, 0, 0, 0, now.Location())
}

func (now *GxTime) EndOfMinute() time.Time {
	return now.BeginningOfMinute().Add(time.Minute - time.Nanosecond)
}

func (now *GxTime) EndOfHour() time.Time {
	return now.BeginningOfHour().Add(time.Hour - time.Nanosecond)
}

func (now *GxTime) EndOfDay() time.Time {
	//return now.BeginningOfDay().Add(24*time.Hour - time.Nanosecond)
	y, m, d := now.Date()
	return time.Date(y, m, d, 23, 59, 59, int(time.Second-time.Nanosecond), now.Location())
}

func (now *GxTime) EndOfWeek() time.Time {
	return now.BeginningOfWeek().AddDate(0, 0, 7).Add(-time.Nanosecond)
}

func (now *GxTime) EndOfMonth() time.Time {
	return now.BeginningOfMonth().AddDate(0, 1, 0).Add(-time.Nanosecond)
}

func (now *GxTime) EndOfQuarter() time.Time {
	return now.BeginningOfQuarter().AddDate(0, 3, 0).Add(-time.Nanosecond)
}

func (now *GxTime) EndOfYear() time.Time {
	return now.BeginningOfYear().AddDate(1, 0, 0).Add(-time.Nanosecond)
}

func (now *GxTime) Monday() time.Time {
	t := now.BeginningOfDay()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	//d := time.Duration(-weekday+1) * 24 * time.Hour
	//return t.Truncate(time.Hour).Add(d)
	return t.AddDate(0, 0, -weekday+1)
}

func (now *GxTime) Sunday() time.Time {
	t := now.BeginningOfDay()
	weekday := int(t.Weekday())
	if weekday == 0 {
		return t
	} else {
		//d := time.Duration(7-weekday) * 24 * time.Hour
		//return t.Truncate(time.Hour).Add(d)
		return t.AddDate(0, 0, (7 - weekday))
	}
}

func (now *GxTime) EndOfSunday() time.Time {
	//return now.Sunday().Add(24*time.Hour - time.Nanosecond)
	return NewGxTime(now.Sunday()).EndOfDay()
}

func parseWithFormat(str string) (t time.Time, err error) {
	for _, format := range TimeFormats {
		t, err = time.Parse(format, str)
		if err == nil {
			return
		}
	}
	err = errors.New("Can't parse string as time: " + str)
	return
}

func (now *GxTime) Parse(strs ...string) (t time.Time, err error) {
	var setCurrentTime bool
	parseTime := []int{}
	currentTime := []int{now.Second(), now.Minute(), now.Hour(), now.Day(), int(now.Month()), now.Year()}
	currentLocation := now.Location()

	for _, str := range strs {
		onlyTime := regexp.MustCompile(`^\s*\d+(:\d+)*\s*$`).MatchString(str) // match 15:04:05, 15

		t, err = parseWithFormat(str)
		location := t.Location()
		if location.String() == "UTC" {
			location = currentLocation
		}

		if err == nil {
			parseTime = []int{t.Second(), t.Minute(), t.Hour(), t.Day(), int(t.Month()), t.Year()}
			onlyTime = onlyTime && (parseTime[3] == 1) && (parseTime[4] == 1)

			for i, v := range parseTime {
				// Don't reset hour, minute, second if it is a time only string
				if onlyTime && i <= 2 {
					continue
				}

				// Fill up missed information with current time
				if v == 0 {
					if setCurrentTime {
						parseTime[i] = currentTime[i]
					}
				} else {
					setCurrentTime = true
				}

				// Default day and month is 1, fill up it if missing it
				if onlyTime {
					if i == 3 || i == 4 {
						parseTime[i] = currentTime[i]
						continue
					}
				}
			}
		}

		if len(parseTime) > 0 {
			t = time.Date(parseTime[5], time.Month(parseTime[4]), parseTime[3], parseTime[2], parseTime[1], parseTime[0], 0, location)
			currentTime = []int{t.Second(), t.Minute(), t.Hour(), t.Day(), int(t.Month()), t.Year()}
		}
	}
	return
}

func (now *GxTime) MustParse(strs ...string) (t time.Time) {
	t, err := now.Parse(strs...)
	if err != nil {
		panic(err)
	}
	return t
}

func (now *GxTime) Between(time1, time2 string) bool {
	restime := now.MustParse(time1)
	restime2 := now.MustParse(time2)
	return now.After(restime) && now.Before(restime2)
}
