package errors

// AppError — обертка для передачи контекста (логирование, код ошибки)
type AppError struct {
    Code    string  // Например: "AUTH_USER_NOT_FOUND"
    Msg     string  // "User not found"
    Err     error   // Первичная причина (sql.ErrNoRows)
    IsFatal bool    // trebaet ли перезагрузки сервиса
}

func (e *AppError) Error() string {
    if e.Msg != "" {
        return e.Msg
    }
    return e.Err.Error()
}

func (e *AppError) Unwrap() error {
    return e.Err
}

// NewAppError — хелпер
func NewAppError(code, msg string, err error) *AppError {
    return &AppError{Code: code, Msg: msg, Err: err}
}