package v1

import "net/http"

func (i *Implementation) UpdateLastSeen(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := i.services.UpdateLastSeen.Execute(r.Context(), userID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
