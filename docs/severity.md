# Severity of Issues

The severity levels for issues are as follows:

* Severity 1

  * **Critical Impact**. This indicates you are unable to use the AWS Private Certificate Authority (PCA) Issuer Plugin in a supported manner, defined in the project [README](../README.md#supported-workflows) and cert-manager [documentation](https://cert-manager.io/docs/), resulting in a critical impact on operations. This condition requires an immediate solution.
  * This can also mean that the PCA Issuer Plugin is unable to operate or has caused other critical software to fail and there is no acceptable way to work around the problem.

* Severity 2

    * **Significant Impact**. This indicates the PCA Issuer Plugin is usable but is severely limited.
    * Severely limited can mean that a [supported workflow](../README.md#supported-workflows) is unable to operate, causes other critical software to fail, or is usable but not without severe difficulty.
    * This can also mean that functionality which you were attempting to use failed, but a temporary work-around is available.
    * This can also mean documentation exists that causes the customer to perform some operation which damages data (unintentional deletion, corruption, etc.).
    * This can include data integrity problems, for example, cases where customer data is inaccurately stored, or retrieved. 

* Severity 3

    * **Some Impact.** This indicates the PCA Issuer Plugin is usable but a [supported workflow](../README.md#supported-workflows) runs with minor issues/limitations.
    * This can also mean the [supported workflow](../README.md#supported-workflows) that you were attempting to use behaved in a manner which is incorrect and/or unexpected, or presented misleading or confusing information.
    * This can include poor or unexplained log messages where no clear error was evident.
    * This can include situations where some side effect is observed which does not significantly harm operations.

* Severity 4

    * **Minimal impact.** This indicates the problem causes little impact on operations or that a reasonable circumvention to the problem has been implemented.
    * This can include incomplete or incorrect documentation.
    * This can also mean the function you were attempting to use suffers from usability quirks, requires minor documentation updates, or could be enhance with some minor changes to the function.
    * This is also the place for general Help/DOC suggestions where data is NOT missing or incorrect.