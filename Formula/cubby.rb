class Cubby < Formula
  desc "Layer profile-scoped dotfiles into a host repo"
  homepage "https://github.com/jmcampanini/cubby"
  head "https://github.com/jmcampanini/cubby.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X github.com/jmcampanini/cubby/cmd.Version=#{version}
    ]
    system "go", "build", "-buildvcs=false", *std_go_args(ldflags:)
    generate_completions_from_executable(bin/"cubby", "completion")
  end

  test do
    assert_match "cubby version HEAD-", shell_output("#{bin}/cubby --version")
  end
end
