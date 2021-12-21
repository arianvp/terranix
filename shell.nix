{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
  name = "shell";
  buildInputs = [ pkgs.go ];
}

