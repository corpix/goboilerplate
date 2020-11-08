{ pkgs ? import ./nixpkgs.nix {} }:
with pkgs;
let
  version      = "fe9d7336a664689eef375914e06dd5c2934d3d3e";
  sha256       = "0zk9qaqcd0mhpbzmi92rmwf4cx8hwlsqlla7h0mjw855fx7r0kaz";
  vendorSha256 = "0mym33jn3jic6fdgp1r8lgndl7gr52f09n3kfhk45y852k5hprs0";
in buildGoModule {
  name = "release-cli";
  src = fetchgit {
    url    = "https://gitlab.com/gitlab-org/release-cli";
    rev    = version;
    sha256 = sha256;
  };

  vendorSha256 = vendorSha256;
  runVend      = true;
  doCheck      = false;

  subPackages = ["cmd/release-cli"];

  buildFlagsArray = [
    "-ldflags=-s -w -X main.VERSION=${version}"
  ];
}
