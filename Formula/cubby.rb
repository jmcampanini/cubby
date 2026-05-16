class Cubby < Formula
  desc "Layer profile-scoped dotfiles into a host repo"
  homepage "https://github.com/jmcampanini/cubby"
  head "https://github.com/jmcampanini/cubby.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/jmcampanini/cubby/cmd.Version=HEAD-#{Utils.git_short_head}"
    system "go", "build", *std_go_args(ldflags: ldflags)
  end

  test do
    assert_match "cubby version HEAD-", shell_output("#{bin}/cubby --version")
  end
end
