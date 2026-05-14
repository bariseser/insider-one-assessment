package request_test

import (
	"encoding/json"

	requestdto "insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Template Request DTOs", func() {
	Describe("CreateTemplateRequest JSON contract", func() {
		It("should marshal expected field names", func() {
			req := requestdto.CreateTemplateRequest{
				Name:    "welcome-email",
				Channel: model.ChannelEmail,
				Content: "hello email",
			}

			payload, err := json.Marshal(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(ContainSubstring(`"name":"welcome-email"`))
			Expect(string(payload)).To(ContainSubstring(`"channel":"email"`))
			Expect(string(payload)).To(ContainSubstring(`"content":"hello email"`))
		})
	})

	Describe("UpdateTemplateRequest JSON contract", func() {
		It("should omit nil fields", func() {
			req := requestdto.UpdateTemplateRequest{}

			payload, err := json.Marshal(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(Equal(`{}`))
		})
	})
})
