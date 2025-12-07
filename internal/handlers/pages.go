package handlers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func IndexPage(c *gin.Context) {
	sess := sessions.Default(c)
	_, ok := sess.Get("user_id").(uint)

	render(c, http.StatusOK, "index.html", gin.H{
		"isAuthed": ok,
	})
}
