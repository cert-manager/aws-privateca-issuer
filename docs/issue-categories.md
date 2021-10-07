# Categories of Issues

The categories of the issues are as follows:

* **Authentication Issue**: This category indicates that you are unable to use authentication methods: [KIAM v4.0+](https://github.com/uswitch/kiam), [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html), or [Kubernetes Secrets](../README.md#authentication). This means that authentication using these methods is broken and therefore severely limits usage of the PCA Issuer Plugin. 

* **Supported Workflow Failure**: This category indicates that a [supported workflow](../README.md#supported-workflows), as defined in the project README, has a critical failure that causes the workflow to be unusable or unable to operate.  

* **Build Issues**: This category indicates that the project fails to build properly or contains build failures. This includes failures that significantly impacts the development environment and prevents modification of the code and/or pull request submission. 

* **Broken Testing Infrastructure**: This category indicates that local or automated testing is broken. This does not include if you have made a modification that is causing test failures. 

* **Incorrect Documentation**: This category indicates that the documentation for the PCA Issuer Plugin is incorrect or incomplete, including typos, broken links, and etc. 

* **Other**: This category indicates that there is an issue that does not fit into any of the categories listed above. 