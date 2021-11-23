let
  # First we setup our overlays. These are overrides of the official nix packages.
  # We do this to pin the versions we want to use of the software that is in
  # the official nixpkgs repo.
  pkgs = import ./nix;
in with pkgs; let
  go-protobuf = buildGoModule rec {
    pname = "go-protobuf";
    version = "v1.5.2";

    src = fetchFromGitHub {
      owner = "golang";
      repo = "protobuf";
      rev = "v1.5.2";
      sha256 = "1mh5fyim42dn821nsd3afnmgscrzzhn3h8rag635d2jnr23r1zhk";
    };

    modSha256 = "0lnk2zpl6y9vnq6h3l42ssghq6iqvmixd86g2drpa4z8xxk116wf";
    vendorSha256 = "1qbndn7k0qqwxqk4ynkjrih7f7h56z1jq2yd62clhj95rca67hh9";

    subPackages = [ "protoc-gen-go" ];
  };
in pkgs.mkShell rec {
  name = "waypoint-plugin-cloudfoundry";

  # The packages in the `buildInputs` list will be added to the PATH in our shell
  buildInputs = [
    pkgs.protobufPin
    go-protobuf
  ];
}
