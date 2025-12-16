# Homebrew formula for popctl
# 
# To use this formula:
# 1. Create a tap repo: github.com/Bidon15/homebrew-tap
# 2. Copy this file to Formula/popctl.rb in that repo
# 3. Update the version and sha256 hashes after each release
# 4. Users install with: brew install Bidon15/tap/popctl

class Popctl < Formula
  desc "POPSigner CLI - manage keys via the control plane API"
  homepage "https://github.com/Bidon15/popsigner"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Bidon15/popsigner/releases/download/popctl-v#{version}/popctl-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"

      def install
        bin.install "popctl-darwin-arm64" => "popctl"
      end
    else
      url "https://github.com/Bidon15/popsigner/releases/download/popctl-v#{version}/popctl-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"

      def install
        bin.install "popctl-darwin-amd64" => "popctl"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/Bidon15/popsigner/releases/download/popctl-v#{version}/popctl-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"

      def install
        bin.install "popctl-linux-arm64" => "popctl"
      end
    else
      url "https://github.com/Bidon15/popsigner/releases/download/popctl-v#{version}/popctl-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"

      def install
        bin.install "popctl-linux-amd64" => "popctl"
      end
    end
  end

  test do
    assert_match "popctl version", shell_output("#{bin}/popctl version")
  end
end

