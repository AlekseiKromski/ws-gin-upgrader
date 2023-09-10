package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"net/http"
)

func JwtCheck(secret string) func(ctx *gin.Context) {
	return func(c *gin.Context) {
		tokenRequest := c.Request.Header.Get("Authorization")
		if tokenRequest == "" || len(tokenRequest) < 10 {
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		tokenRequest = tokenRequest[7:len(tokenRequest)]

		token, err := jwt.Parse(tokenRequest, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("wrong sign method")
			}
			claims := token.Claims.(jwt.MapClaims)
			if claims["id"] == nil {
				return nil, fmt.Errorf("wrong format of JWT: %s", claims)
			}

			uid := claims["id"]
			c.Set("uid", uid)

			return secret, nil
		})

		if err != nil {
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		if !token.Valid {
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
