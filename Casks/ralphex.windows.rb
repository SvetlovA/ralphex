# Updated by GoReleaser's Homebrew cask PR after each release.
cask "ralphex.windows" do
  version "1.7.0-windows.2"

  on_macos do
    on_arm do
      sha256 "68ba1350b58e51f7764b57c6cf1f88d27da2bb0d9b110059f937bfb59a027581"
      url "https://github.com/SvetlovA/ralphex/releases/download/v#{version}/ralphex_#{version}_darwin_arm64.tar.gz"
    end
    on_intel do
      sha256 "35df893c30c64d92a8770900039aee3e2fd686b579bdcc020d7bc3daeae2c7d9"
      url "https://github.com/SvetlovA/ralphex/releases/download/v#{version}/ralphex_#{version}_darwin_amd64.tar.gz"
    end
  end

  on_linux do
    on_arm do
      sha256 "3c87ad6bcabd35324d1255c11e98f96569cd706ea958d191675e5231257a2173"
      url "https://github.com/SvetlovA/ralphex/releases/download/v#{version}/ralphex_#{version}_linux_arm64.tar.gz"
    end
    on_intel do
      sha256 "dbb21391d321f8feb2f8d49d3c001b79878f9e42356f211e6fc4c36fd58777e5"
      url "https://github.com/SvetlovA/ralphex/releases/download/v#{version}/ralphex_#{version}_linux_amd64.tar.gz"
    end
  end

  name "ralphex.windows"
  desc "Autonomous plan execution with Claude Code"
  homepage "https://github.com/SvetlovA/ralphex"

  binary "ralphex"
  bash_completion "completions/ralphex.bash"
  zsh_completion "completions/ralphex.zsh"
  fish_completion "completions/ralphex.fish"

  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/ralphex"]
    end
  end
end
