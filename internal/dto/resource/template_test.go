package resource_test

import (
	"encoding/json"
	"time"

	resourcedto "insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Template Resource DTOs", func() {
	It("should marshal template resource", func() {
		now := time.Date(2026, time.May, 13, 20, 0, 0, 0, time.UTC)
		resp := resourcedto.TemplateResource{
			ID:        "template-1",
			Name:      "welcome-email",
			Channel:   model.ChannelEmail,
			Content:   "hello email",
			CreatedAt: now,
			UpdatedAt: now,
		}

		payload, err := json.Marshal(resp)

		Expect(err).NotTo(HaveOccurred())
		Expect(string(payload)).To(ContainSubstring(`"id":"template-1"`))
		Expect(string(payload)).To(ContainSubstring(`"name":"welcome-email"`))
		Expect(string(payload)).To(ContainSubstring(`"channel":"email"`))
		Expect(string(payload)).To(ContainSubstring(`"content":"hello email"`))
	})
})
