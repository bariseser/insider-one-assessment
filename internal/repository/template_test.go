package repository_test

import (
	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/repository"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Template Repository", func() {
	Describe("Template CRUD", func() {
		It("should create, fetch, list, update, and delete a template", func() {
			created := model.MessageTemplate{
				Name:    "welcome-sms",
				Channel: model.ChannelSMS,
				Content: "welcome from template",
			}
			err := templateRepository.CreateTemplate(ctx, &created)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeEmpty())

			fetched, err := templateRepository.GetTemplateByID(ctx, created.ID.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(fetched.Name).To(Equal("welcome-sms"))
			Expect(fetched.Channel).To(Equal(model.ChannelSMS))
			Expect(fetched.Content).To(Equal("welcome from template"))

			templates, err := templateRepository.ListTemplates(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(templates).To(HaveLen(1))
			Expect(templates[0].ID).To(Equal(created.ID))
			Expect(templates[0].Channel).To(Equal(model.ChannelSMS))

			created.Name = "welcome-sms-v2"
			created.Content = "updated template content"
			err = templateRepository.UpdateTemplate(ctx, &created)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Name).To(Equal("welcome-sms-v2"))
			Expect(created.Content).To(Equal("updated template content"))
		})

		It("should return conflict when creating a template with a duplicate name", func() {
			first := model.MessageTemplate{
				Name:    "duplicate-template",
				Channel: model.ChannelSMS,
				Content: "first",
			}
			err := templateRepository.CreateTemplate(ctx, &first)
			Expect(err).NotTo(HaveOccurred())

			second := model.MessageTemplate{
				Name:    "duplicate-template",
				Channel: model.ChannelEmail,
				Content: "second",
			}
			err = templateRepository.CreateTemplate(ctx, &second)
			Expect(err).To(MatchError(repository.ErrTemplateAlreadyExists))
		})
	})
})
