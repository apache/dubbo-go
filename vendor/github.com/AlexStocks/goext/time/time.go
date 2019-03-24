// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxtime encapsulates some golang.time functions
package gxtime

import (
	"strconv"
	"time"
)

func TimeDayDuratioin(day float64) time.Duration {
	return time.Duration(day * 24 * float64(time.Hour))
}

func TimeHourDuratioin(hour float64) time.Duration {
	return time.Duration(hour * float64(time.Hour))
}

func TimeMinuteDuration(minute float64) time.Duration {
	return time.Duration(minute * float64(time.Minute))
}

func TimeSecondDuration(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

func TimeMillisecondDuration(m float64) time.Duration {
	return time.Duration(m * float64(time.Millisecond))
}

func TimeMicrosecondDuration(m float64) time.Duration {
	return time.Duration(m * float64(time.Microsecond))
}

func TimeNanosecondDuration(n float64) time.Duration {
	return time.Duration(n * float64(time.Nanosecond))
}

// desc: convert year-month-day-hour-minute-seccond to int in second
// @month: 1 ~ 12
// @hour:  0 ~ 23
// @minute: 0 ~ 59
func YMD(year int, month int, day int, hour int, minute int, sec int) int {
	return int(time.Date(year, time.Month(month), day, hour, minute, sec, 0, time.Local).Unix())
}

// @YMD in UTC timezone
func YMDUTC(year int, month int, day int, hour int, minute int, sec int) int {
	return int(time.Date(year, time.Month(month), day, hour, minute, sec, 0, time.UTC).Unix())
}

func YMDPrint(sec int, nsec int) string {
	return time.Unix(int64(sec), int64(nsec)).Format("2006-01-02 15:04:05.99999")
}

func Future(sec int, f func()) {
	time.AfterFunc(TimeSecondDuration(float64(sec)), f)
}

func Unix2Time(unix int64) time.Time {
	return time.Unix(unix, 0)
}

func UnixNano2Time(nano int64) time.Time {
	return time.Unix(nano/1e9, nano%1e9)
}

func UnixString2Time(unix string) time.Time {
	i, err := strconv.ParseInt(unix, 10, 64)
	if err != nil {
		panic(err)
	}

	return time.Unix(i, 0)
}

// 注意把time转换成unix的时候有精度损失，只返回了秒值，没有用到纳秒值
func Time2Unix(t time.Time) int64 {
	return t.Unix()
}

func Time2UnixNano(t time.Time) int64 {
	return t.UnixNano()
}

func GetEndtime(format string) time.Time {
	timeNow := time.Now()
	switch format {
	case "day":
		year, month, _ := timeNow.Date()
		nextDay := timeNow.AddDate(0, 0, 1).Day()
		t := time.Date(year, month, nextDay, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)

	case "week":
		year, month, _ := timeNow.Date()
		weekday := int(timeNow.Weekday())
		weekendday := timeNow.AddDate(0, 0, 8-weekday).Day()
		t := time.Date(year, month, weekendday, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)

	case "month":
		year := timeNow.Year()
		nextMonth := timeNow.AddDate(0, 1, 0).Month()
		t := time.Date(year, nextMonth, 1, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)

	case "year":
		nextYear := timeNow.AddDate(1, 0, 0).Year()
		t := time.Date(nextYear, 1, 1, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)
	}

	return timeNow
}
