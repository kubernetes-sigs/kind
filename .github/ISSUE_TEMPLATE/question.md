---
name: Question
about: Ask a question about using kind
labels: kind/support

---

<!-- Consider also checking https://kind.sigs.k8s.io/#community-discussion-contribution-and-support for support, our slack community is especially helpful! -->

<!--
If your question is about a cluster failing to start, please attach the cluster logs:

1. Run the same create command again with `--retain`, so the nodes are not cleaned up on failure:
   `kind create cluster --retain <your other flags>`
2. Export the logs to a directory (add `--name <cluster-name>` if you used a non-default name):
   `kind export logs ./kind-logs`
3. Archive the directory: `tar -czf kind-logs.tar.gz ./kind-logs`
4. Attach `kind-logs.tar.gz` to this issue by dragging it into this text box (GitHub cannot upload a directory).
5. Clean up: `kind delete cluster`
-->
