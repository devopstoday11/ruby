package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testThin(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building a thin app", func() {
		var (
			image     occam.Image
			container occam.Container

			name string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		})

		it("creates a working OCI image with a thin start command", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithBuildpacks(rubyBuildpack).
				WithNoPull().
				Execute(name, filepath.Join("testdata", "thin"))
			Expect(err).NotTo(HaveOccurred(), logs.String())

			container, err = docker.Container.Run.WithEnv(map[string]string{"PORT": "8080"}).Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable(), ContainerLogs(container.ID))

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello world!"))

			Expect(logs).To(ContainLines(ContainSubstring("MRI Buildpack")))
			Expect(logs).To(ContainLines(ContainSubstring("Bundler Buildpack")))
			Expect(logs).To(ContainLines(ContainSubstring("Bundle Install Buildpack")))
			Expect(logs).To(ContainLines(ContainSubstring("Thin Buildpack")))
		})
	})
}
