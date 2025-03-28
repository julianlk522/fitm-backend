package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/go-chi/render"
	e "github.com/julianlk522/fitm/error"
)

func HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	signkey_secret := os.Getenv("FITM_WEBHOOK_SECRET")
	if signkey_secret == "" {
		render.Render(w, r, e.Err500(e.ErrNoWebhookSecret))
		return
	}

	signature_header := r.Header.Get("X-Hub-Signature-256")
	if signature_header == "" || !strings.HasPrefix(signature_header, "sha256=") {
		render.Render(w, r, e.ErrUnauthorized(e.ErrNoWebhookSignature))
		return
	}

	// get signature, skipping "sha256="
	gh_hash := signature_header[7:]

	// get payload
	defer r.Body.Close()
	payload_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Print("Cannot read GH webhook request payload")
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	// generate new hmac using secret
	server_hash := hmac.New(
		sha256.New,
		[]byte(signkey_secret),
	)
	// update hash object with payload
	if _, err := server_hash.Write(payload_bytes); err != nil {
		log.Print("Cannot compute HMAC for GH webhook request body")
		render.Render(w, r, e.Err500(err))
		return
	}

	// generate expected signature
	server_signature := "sha256=" + hex.EncodeToString(server_hash.Sum(nil))
	if !hmac.Equal([]byte(gh_hash), []byte(server_signature)) {
		render.Render(w, r, e.ErrUnauthorized(e.ErrInvalidWebhookSignature))
		return
	}

	log.Println("Authenticated webhook: updating FITM backend")

	fitm_root_path := os.Getenv("FITM_BACKEND_ROOT")
	cmd := exec.Command("./update_and_restart_backend.sh")
	cmd.Dir = fitm_root_path
	err = cmd.Run()
	if err != nil {
		log.Println(err)
		render.Render(w, r, e.Err500(err))
		return
	}

	render.Status(r, http.StatusOK)
}
