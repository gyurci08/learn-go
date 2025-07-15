package db

// Define the HelloWorld model
type HelloWorld struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Message string `gorm:"type:varchar(255)" json:"message"`
}

// Rename the table
func (HelloWorld) TableName() string {
	return "hello_worlds"
}
