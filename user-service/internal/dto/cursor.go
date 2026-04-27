package dto

import (
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"
)

// CursorUsernamePayload — данные внутри курсора для keyset-пагинации по (username, id).
// Сериализуется в base64-JSON и передаётся клиенту как непрозрачная строка.
type CursorUsernamePayload struct {
	Username string    `json:"username"`
	ID       uuid.UUID `json:"id"`
}

// EncodeCursor сериализует курсор в base64-строку для передачи клиенту.
func EncodeCursor(c CursorUsernamePayload) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DecodeCursor десериализует курсор из base64-строки, полученной от клиента.
func DecodeCursor(s string) (CursorUsernamePayload, error) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return CursorUsernamePayload{}, err
	}
	var c CursorUsernamePayload
	if err := json.Unmarshal(b, &c); err != nil {
		return CursorUsernamePayload{}, err
	}
	return c, nil
}
