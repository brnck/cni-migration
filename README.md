cni-migration is a CLI tool for migrating a Kubernetes cluster's CNI solution
from AWS VPC CNI to Cilium. The tool works in pre-migration and post-migration way.
Pre-migration prepares nodes and Kubernetes environment in general for the new CNI to be deployed.
Then we can use EKS managed node groups to spin up new nodes with particular label to ensure
only Cilium CNI will be deployed to those nodes. Also, `knet-stress` daemon set is used to ensure
all nodes have communication during the whole migration process

## Disclaimer

This repository is a fork of [jetstack/cni-migration](https://github.com/jetstack/cni-migration) which
helps to migrate CNI from Canal to Cilium and use more general approach. Since, we are migrating from AWS VPC CNI
to Cilium and they both use AWS VPC to manage ip addresses, we are using slightly different approach 

## How

The following are the steps taken to migrate the CNI. During and after each
step, the inter-pod communication is regularly tested using
[knet-stress](https://github.com/jetstack/knet-stress), which will send a HTTP
request to all other knet-stress instances on all nodes. This proves a
bi-directional network connectivity across cluster.

### Pre-migration

0. This step deploys knet-stress and ensure each pod of can "talk" to each other
1. This step disables cluster autoscaler by descaling deployment to 0. This will ensure no new nodes
   are being created until the migration is finished.
2. This step will label all nodes with `node-role.kubernetes/aws-vpc=true`.
3. This step will add node selector to AWS VPC CNI daemon set to ensure pods will be scheduled only
   in nodes that contain label `node-role.kubernetes/aws-vpc=true`.
4. This step will deploy `Cilium` to the cluster. The daemonset of Cilium already has node-selector set
   so pods will not be scheduled unless node has label `node-role.kubernetes/cilium=true`. Other dependencies
   will be deployed and scheduled as is

<...> 

### Post-migration

5. This step will remove `aws-node` daemon set from the cluster to ensure there are no two CNIs in the cluster
6. This step will remove label `node-role.kubernetes/aws-vpc=true` from the nodes
7. This step will update Cilium by removing node selector `node-role.kubernetes/cilium=true`
8. This step will remove label `node-role.kubernetes/cilium=true` from the nodes
9. This step will re-enable cluster autoscaler by setting replicas number that is set in 
   config.yaml under the `clusterAutoscaler.replicas` key 

The cluster should now be fully migrated from AWS VPC CNI to Cilium CNI.

## Configuration

The cni-migration tool has input configuration file (default `--config
conifg.yaml`), that holds options for the migration.

### labels

This holds options on which label keys and shared value should be used for each
signal of steps:

```yaml
  aws-vpc-cni: node-role.kubernetes.io/aws-vpc
  cilium: node-role.kubernetes.io/cilium
  value: "true" # used as the value to each label key
```

### paths

The file paths for each manifest bundle:

```yaml
  knet-stress: ./resources/knet-stress.yaml
  cilium-pre-migration: ./resources/cilium-pre-migration.yaml
  cilium-post-migration: ./resources/cilium-post-migration.yaml
```

### cilium

Cilium helm chart release configuration:

```yaml
  release-name: cilium
  chart-name: cilium/cilium
  repo-path: "https://helm.cilium.io/"
  version: 1.12.5
  namespace: kube-system
```
