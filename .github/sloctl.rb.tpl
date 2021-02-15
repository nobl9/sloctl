class Sloctl < Formula
    desc "Command-line client for Nobl9"
    homepage "https://nobl9.com"
    version "${SLOCTL_VERSION}"
    bottle :unneeded
  
    if OS.mac?
      url "https://github.com/nobl9/sloctl/releases/download/${SLOCTL_VERSION}/sloctl-macos-${SLOCTL_VERSION}.zip"
      sha256 "${SHA_MACOS}"
    elsif OS.linux?
      url "https://github.com/nobl9/sloctl/releases/download/${SLOCTL_VERSION}/sloctl-linux-${SLOCTL_VERSION}.zip"
      sha256 "${SHA_LINUX}"
    end
  
    def install
      bin.install "sloctl"
    end
  
    def caveats
        <<~EOS
          Thank you for installing the command-line client for Nobl9!
    
          To see help and a list of available commands type:
            $ sloctl help 
   
          For more information on how to use the command-line client
          and the Nobl9 managed cloud service, visit:
            https://nobl9.com
        EOS
      end

    test do
        assert_predicate bin/"sloctl", :exist?
        system "sloctl", "--help"
      end
  end
