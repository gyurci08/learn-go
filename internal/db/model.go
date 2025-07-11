package db

type HelloWorld struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Message string `gorm:"type:varchar(255)" json:"message"`
}
