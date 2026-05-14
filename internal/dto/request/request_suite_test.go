package request_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDTOs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DTO Request Suite")
}
