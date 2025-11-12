package model

// Member represents a user in the system
// Oracle sequence MEMBER_SEQ is used for ID generation
type Member struct {
	// Primary key - Oracle IDENTITY (auto-increment)
	ID uint32 `gorm:"column:id;primaryKey;autoIncrement"`

	// Core fields
	Email       string `gorm:"column:email;type:VARCHAR2(255);not null;uniqueIndex:idx_member_email"` // 이메일 (unique)
	Name        string `gorm:"column:name;type:VARCHAR2(100);not null"`                               // 이름
	PhoneNumber string `gorm:"column:phone_number;type:VARCHAR2(100);not null"`                       // 핸드폰 번호
	Password    string `gorm:"column:password;type:VARCHAR2(60);not null"`                            // 암호화된 비밀번호

	BaseEntity
}

// TableName specifies the table name for Member
func (*Member) TableName() string {
	return "member"
}

// NewMember creates a new Member instance with validation
// Factory method pattern (Java의 static create 메서드와 동일)
func NewMember(name, email, phoneNumber, password string) *Member {
	// Note: password should be hashed before storing (handled in service layer)
	return &Member{
		Name:        name,
		Email:       email,
		PhoneNumber: phoneNumber,
		Password:    password, // This should be hashed password
	}
}
