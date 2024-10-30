package main

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	_ "github.com/sijms/go-ora/v2"
)

type Employee struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	LName        string `json:"lname"`
	Nlock        int    `json:"nlock"`
	Sex          string `json:"sex"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	DeptID       int    `json:"dept_id"`
	RoleID       int    `json:"role_id"`
	ProfileImage []byte `json:"profile_image"`
}

func createEmployeeInDB(employee *Employee) error {
	var id int
	// Check if an email exists
	query := `SELECT email FROM employee WHERE email=:1`
	err := db.QueryRow(query, employee.Email).Scan(&id)
	if err != sql.ErrNoRows {
		return fiber.ErrConflict
	}
	// Execute if no email exists
	err = db.QueryRow("SELECT max(id) FROM employee").Scan(&id)
	if err != nil {
		return err
	}
	query = `
    INSERT INTO employee (id, name, lname, nlock, sex, email, password, dept_id, role_id)
    VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9)
	`
	_, err = db.Exec(query, id+1, employee.Name, employee.LName, 0, employee.Sex, employee.Email, employee.Password, employee.DeptID, 2)
	if err != nil {
		return err
	}
	return nil
}

func registerHandler(c *fiber.Ctx) error {
	employee := new(Employee)
	err := c.BodyParser(&employee)
	if err != nil {
		return err
	}
	err = createEmployeeInDB(employee)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"message": "Register Successfully",
	})
}