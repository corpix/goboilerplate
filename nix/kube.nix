let
  defaultNamespace = "goboilerplate";
in { nixpkgsPath           ? <nixpkgs>
   , pkgs                  ? import nixpkgsPath {}
   , lib                   ? pkgs.lib
   , system                ? builtins.currentSystem

   , kubenixPath           ? fetchGit { url = "https://github.com/blaggacao/kubenix";
                                        rev = "a0ce293db80c335eb35c89786ae63f2c2160bab8";
                                      }
   , kubenix               ? import kubenixPath {}
   , tools                 ? lib.mergeAttrs
     (import "${kubenixPath}/lib/k8s" { inherit lib; })
     (import "${kubenixPath}/lib/upstreamables.nix" { inherit pkgs lib; })
   , evalModules           ? kubenix.evalModules.${system}

   , name                  ? "goboilerplate"
   , namespace             ? defaultNamespace
   , deployment            ? name
   , labels                ? { app = name;
                               deployment = name;
                             }
   , replicas              ? 2

   , container             # required (ex: import ./container.nix {})
   , hostname              # required (ex: localhost)
   , routeHostname         ? hostname
   , port                  ? 4180 # FIXME: fill your app port
   , portProtocol          ? "tcp"
   , telemetryPort         ? 4280
   , telemetryPortProtocol ? "tcp"
   , limits                ? { cpu = "256m"; memory = "256Mi"; }

   , configMntPrefix       ? "/etc/goboilerplate"
   , secretsMntPrefix      ? "/secrets"
   , secrets               ? {}
   , ... }:
with builtins;
with lib;
let
  mkSecretPath = secretName: "${secretsMntPrefix}/${secretName}";
  mkMetaLabels = baseLabels: labels // {
    "app.kubernetes.io/part-of" = labels.deployment;
  };

  ##

  mkServiceConfig =
    { ... }: {
      log.level = "info";
      telemetry = {
        enable = true;
        path = "/metrics";
        addr = "0.0.0.0:${toString telemetryPort}";
      };
    };

  mkConfigMaps = {name, labels, hostname, ...} @ args: {
    "${name}-config" = {
      metadata = {
        name = "${name}-config";
        labels = mkMetaLabels labels;
      };
      data."config.yml" = toJSON (mkServiceConfig args);
    };
  };

  mkSecrets = {name, labels, secrets, ...}: {
    "${name}-secrets" = {
      metadata = {
        name = "${name}-secrets";
        labels = mkMetaLabels labels;
      };
      data = mapAttrs (name: value: tools.toBase64 value) secrets;
    };
  };

  mkDeployments = {config, name, labels, replicas, ...} @ args: {
    ${name} = {
      metadata = {
        inherit name;
        labels = mkMetaLabels labels;
      };
      spec = {
        inherit replicas;

        selector.matchLabels = labels;
        template = {
          metadata.labels = mkMetaLabels labels;
          spec = {
            # FIXME: should be OnFailure, but openshift:
            # Unsupported value: "OnFailure": supported values: "Always"
            restartPolicy = "Always";

            terminationGracePeriodSeconds = 30;

            volumes = {
              "${name}-config".configMap.name = "${name}-config";
              "${name}-secrets".secret.secretName = "${name}-secrets";
            };

            containers.${name} = rec {
              image = config.docker.images.${name}.path;
              imagePullPolicy = "IfNotPresent";

              volumeMounts = [
                { name = "${name}-config";
                  mountPath = configMntPrefix;
                  readOnly = true;
                }
                { name = "${name}-secrets";
                  mountPath = secretsMntPrefix;
                  readOnly = true;
                }
              ];

              ports = {
                "${toString port}" = { inherit name; };
                "4280" = { name = "metrics"; };
              };

              env = [
                # NOTE: read-only filesystem inside container,
                # no need to bother with pid-files
                { name = "GOBOILERPLATE_PID_FILE";
                  value = "";
                }
              ];

              securityContext = {
                readOnlyRootFilesystem = true;
                runAsNonRoot = true;
              };
              resources = {
                inherit limits;
                requests = limits;
              };
              # livenessProbe.httpGet = {
              #   path = "/ping";
              #   port = name;
              # };

              # readinessProbe = livenessProbe;
            };
          };
        };
      };
    };
  };

  mkServices = {config, name, labels, ...} @ args: {
    ${name} = {
      metadata = {
        inherit name;
        labels = mkMetaLabels labels;
      };
      spec = let
        mkClusterIP = port: protocol: {
          name = "${toString port}-${protocol}";
          protocol = "${toUpper protocol}";
          port = port;
          targetPort = port;
        };
      in {
        type = "ClusterIP";
        ports = [
          (mkClusterIP port portProtocol)
          (mkClusterIP telemetryPort telemetryPortProtocol)
        ];
        selector = labels;
      };
    };
  };

  mkRoutes = {config, name, labels, hostname, ...} @ args: {
    ${name} = {
      metadata = {
        inherit name;
        labels = mkMetaLabels labels;
      };
      spec = {
        host = hostname;
        to = {
          inherit name;
          kind = "Service";
          weight = 100;
        };
        port.targetPort = "${toString port}-${portProtocol}";
        tls.termination = "edge";
      };
    };
  };

  ##

  mkManifest = config: let
    apply = fn: args: fn ({ inherit config name labels secrets; } // args);
  in {
    inherit namespace;

    resources.namespaces.namespace = mkIf (defaultNamespace == namespace) {
      kind = "Namespace";
      metadata.name = namespace;
    };

    resources = {
      configMaps = apply mkConfigMaps { inherit hostname; };
      secrets = apply mkSecrets {};
      deployments = apply mkDeployments { inherit replicas; };
      services = apply mkServices {};
      routes = apply mkRoutes { hostname = routeHostname; };
    };

    customTypes = [
      { # TODO: typing for openshift routes is incomplete, but leave this for now
        attrName = "routes";
        group    = "route.openshift.io";
        version  = "v1";
        kind     = "Route";


        module.options.metadata = mkOption { type = types.attrs; };
        module.options.spec = mkOption { type = types.attrs; };
      }
    ];
  };

  ##

  modules = evalModules {
    modules = [
      ({ config, kubenix, ... }: {
        imports = with kubenix.modules; [k8s docker];

        config.kubernetes = mkManifest config;
        config.docker.images.${name}.image = container;
      })
    ];
  };
in {
  inherit (modules) config;

  # manifest file contents
  manifest = tools.mkHashedList { items = modules.config.kubernetes.objects; };

  # manifest location inside /nix/store
  manifestPath = config.kubernetes.result;
}
