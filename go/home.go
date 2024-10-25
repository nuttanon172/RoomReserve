package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/sijms/go-ora/v2"
)

func formatTime(t string) string {
	if len(t) == 4 {
		t = "0" + t
	}
	t = strings.Replace(t, ".", ":", 1)
	return t
}

func getHomeHandler(c *fiber.Ctx) error {
	fmt.Println("Function home called")
	var rooms []map[string]interface{}
	var params []interface{}

	var placeholderIndex int

	selectedBuilding := c.Query("building", "")
	selectedFloor := c.Query("floor", "")
	selectedRoom := c.Query("room", "")
	selectedType := c.Query("type", "")
	selectedPeople := c.Query("people", "")
	selectedDate := c.Query("date", "")
	begin := c.Query("time", "")
	end := c.Query("time2", "")

	selectedTime := formatTime(begin)
	selectedTime2 := formatTime(end)

	query := `
    SELECT DISTINCT r.id, r.name, r.description, r.status, r.cap, r.room_type_id, f.name, b.name, rt.name
    FROM room r
    JOIN room_type rt ON r.room_type_id = rt.id
    JOIN building_floor bf ON r.address_id = bf.id
    JOIN FLOOR f ON f.id = bf.floor_id
    JOIN building b ON b.id = bf.building_id`

	if selectedTime != "" && selectedTime2 != "" && selectedDate != "" {
		query += `
	LEFT JOIN BOOKING book ON r.id = book.room_id 
	AND TRUNC(book.start_time) = TO_DATE(:` + strconv.Itoa(placeholderIndex+1) + `, 'YYYY-MM-DD')
	AND (
		(:` + strconv.Itoa(placeholderIndex+2) + ` BETWEEN TO_CHAR(book.start_time, 'HH24:MI') AND TO_CHAR(book.end_time, 'HH24:MI')) 
		OR (:` + strconv.Itoa(placeholderIndex+3) + ` BETWEEN TO_CHAR(book.start_time, 'HH24:MI') AND TO_CHAR(book.end_time, 'HH24:MI')) 
		OR (TO_CHAR(book.start_time, 'HH24:MI') BETWEEN :` + strconv.Itoa(placeholderIndex+2) + ` AND :` + strconv.Itoa(placeholderIndex+3) + `)
		OR (TO_CHAR(book.end_time, 'HH24:MI') BETWEEN :` + strconv.Itoa(placeholderIndex+2) + ` AND :` + strconv.Itoa(placeholderIndex+3) + `)
		OR (TO_CHAR(book.start_time, 'HH24:MI') < :` + strconv.Itoa(placeholderIndex+2) + ` AND TO_CHAR(book.end_time, 'HH24:MI') > :` + strconv.Itoa(placeholderIndex+3) + `)
		OR (:` + strconv.Itoa(placeholderIndex+2) + ` < TO_CHAR(book.end_time, 'HH24:MI') AND :` + strconv.Itoa(placeholderIndex+3) + ` > TO_CHAR(book.start_time, 'HH24:MI'))
		OR (:` + strconv.Itoa(placeholderIndex+3) + ` >= TO_CHAR(book.end_time, 'HH24:MI') AND :` + strconv.Itoa(placeholderIndex+2) + ` < TO_CHAR(book.start_time, 'HH24:MI'))

	)`
		query += `
			WHERE book.room_id IS NULL`

		params = append(params, selectedDate, selectedTime, selectedTime2)

		placeholderIndex += 3

	} else {
		query += " WHERE 1 = 1 "
	}
	if selectedBuilding != "" {
		query += " AND b.name = :" + strconv.Itoa(placeholderIndex+1)
		params = append(params, selectedBuilding)
		placeholderIndex++
	}
	if selectedFloor != "" {
		query += " AND f.name = :" + strconv.Itoa(placeholderIndex+1)
		params = append(params, selectedFloor)
		placeholderIndex++
	}
	if selectedRoom != "" {
		query += " AND r.name = :" + strconv.Itoa(placeholderIndex+1)
		params = append(params, selectedRoom)
		placeholderIndex++
	}
	if selectedType != "" {
		query += " AND rt.name = :" + strconv.Itoa(placeholderIndex+1)
		params = append(params, selectedType)
		placeholderIndex++
	}
	if selectedPeople != "" {
		cap, err := strconv.Atoi(selectedPeople)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid people count"})
		}
		query += " AND r.cap >= :" + strconv.Itoa(placeholderIndex+1)
		params = append(params, cap)
		placeholderIndex++
	}

	rows, err := db.Query(query, params...)
	if err != nil {
		fmt.Println("Error fetching rooms:", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	defer rows.Close()

	for rows.Next() {
		var id, status, cap, roomTypeID int
		var name, description, typeName, floor, building string

		if err := rows.Scan(&id, &name, &description, &status, &cap, &roomTypeID, &floor, &building, &typeName); err != nil {
			fmt.Println("Error scanning room:", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rooms = append(rooms, map[string]interface{}{
			"id":           id,
			"name":         name,
			"description":  description,
			"status":       status,
			"cap":          cap,
			"room_type_id": roomTypeID,
			"floor":        floor,
			"building":     building,
			"type_name":    typeName,
		})
	}
	if len(rooms) == 0 {
		suggestion := getRoomSuggestion(selectedDate, selectedTime, selectedTime2)
		return c.JSON(fiber.Map{
			"message":    "ไม่พบห้องว่างในช่วงเวลาที่เลือก",
			"suggestion": suggestion,
		})
	}

	return c.JSON(rooms)
}

func getRoomSuggestion(date, startTime, endTime string) string {
	layout := "15:04" // รูปแบบเวลา
	datelayout := "2006-01-02"
	startTimeParsed, err := time.Parse(layout, formatTime(startTime))
	if err != nil {
		return "เกิดข้อผิดพลาดในการแปลงเวลา"
	}
	endTimeParsed, err := time.Parse(layout, formatTime(endTime))
	if err != nil {
		return "เกิดข้อผิดพลาดในการแปลงเวลาจบ"
	}
	datetimeParsed, err := time.Parse(datelayout, date)
	if err != nil {
		return "เกิดข้อผิดพลาดในการแปลงวัน"
	}
	diff := endTimeParsed.Sub(startTimeParsed)
	var i int

	for {
		// สร้างเวลาที่มีปีและวันที่ถูกต้องเสมอ
		newStartTime := time.Date(datetimeParsed.Year(), datetimeParsed.Month(), datetimeParsed.Day(), startTimeParsed.Hour(), startTimeParsed.Minute(), 0, 0, time.Local).Add(time.Duration(i) * time.Hour)
		newEndTime := newStartTime.Add(diff)

		newStartTimeStr := newStartTime.Format(layout)
		newEndTimeStr := newEndTime.Format(layout)
		newdateStr := newStartTime.Format(datelayout)

		roomAvailable := checkRoomAvailability(newdateStr, newStartTimeStr, newEndTimeStr)
		if !roomAvailable {
			return fmt.Sprintf("เวลาแนะนำที่ใกล้เคียงที่สุดคือในวันที่ %s เวลา %s - %s", newdateStr, newStartTimeStr, newEndTimeStr)
		}

		if newEndTime.Hour() >= 18 {
			// เปลี่ยนวัน
			datetimeParsed = datetimeParsed.AddDate(0, 0, 1)
			i = 0
			continue
		}

		i++
	}

}

func checkRoomAvailability(date, startTime, endTime string) bool {
	if date == "" || startTime == "" || endTime == "" {
		fmt.Println("ค่าตัวแปรไม่ถูกต้อง:", date, startTime, endTime)
		return false
	}
	var params []interface{}

	start := formatTime(startTime)
	end := formatTime(endTime)

	query := `SELECT COUNT(*) 
	FROM room r 
	LEFT JOIN BOOKING book ON r.id = book.room_id 
	AND TRUNC(book.start_time) = TO_DATE( :1, 'YYYY-MM-DD')
	AND (
		(:2 BETWEEN TO_CHAR(book.start_time, 'HH24:MI') AND TO_CHAR(book.end_time, 'HH24:MI')) 
		OR (:3 BETWEEN TO_CHAR(book.start_time, 'HH24:MI') AND TO_CHAR(book.end_time, 'HH24:MI')) 
		OR (TO_CHAR(book.start_time, 'HH24:MI') BETWEEN :2 AND :3)
		OR (TO_CHAR(book.end_time, 'HH24:MI') BETWEEN :2 AND :3)
		OR (TO_CHAR(book.start_time, 'HH24:MI') < :2 AND TO_CHAR(book.end_time, 'HH24:MI') > :3)
		OR (:2 < TO_CHAR(book.end_time, 'HH24:MI') AND :3 > TO_CHAR(book.start_time, 'HH24:MI'))
		OR (:3 >= TO_CHAR(book.end_time, 'HH24:MI') AND :2 < TO_CHAR(book.start_time, 'HH24:MI'))
		

	)WHERE book.room_id IS NULL`

	var count int
	params = append(params, date, start, end)

	err := db.QueryRow(query, params...).Scan(&count)
	if err != nil {
		fmt.Println("Error checking room availability:", err)
		return false
	}

	return count == 0
}
