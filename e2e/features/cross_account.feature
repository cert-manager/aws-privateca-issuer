@CrossAccount
Feature: Issue certificates using a CA in another account 
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates from a CA in another account

  @AWSPCAClusterIssuer
  Scenario Outline: Issue a certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | certType |
      | RSA      |
      | ECDSA    |

  @AWSPCAClusterIssuer
  Scenario Outline: Issue a CA certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a XA CA
    And I update my certificate spec to issue a CA certificate
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | certType |
      | RSA      |
      | ECDSA    |

  @AWSPCAClusterIssuer
  Scenario Outline: Issue a short lived certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    And I update my certificate spec to issue a certificate with duration of 20 hours
    And I update my certificate spec to issue a certificate with renew before of 5 hours
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | certType |
      | RSA      |
      | ECDSA    |

  @AWSPCAClusterIssuer @KubernetesSecrets   
  Scenario Outline: Issue a certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAClusterIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA    |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | myKeyId           | mySecret              | RSA      |
      | myKeyId           | mySecret              | ECDSA    |

  @AWSPCAClusterIssuer @KubernetesSecrets
  Scenario Outline: Issue a CA certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAClusterIssuer using a XA CA
    And I update my certificate spec to issue a CA certificate
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA    |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | myKeyId           | mySecret              | RSA      |
      | myKeyId           | mySecret              | ECDSA    |

  @AWSPCAClusterIssuer @KubernetesSecrets
  Scenario Outline: Issue a short lived certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAClusterIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    And I update my certificate spec to issue a certificate with duration of 20 hours
    And I update my certificate spec to issue a certificate with renew before of 5 hours
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA    |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | myKeyId           | mySecret              | RSA      |
      | myKeyId           | mySecret              | ECDSA    |

  @AWSPCAIssuer
  Scenario Outline: Issue a certificate with a namespace issuer 
    Given I create an AWSPCAIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | certType |
      | RSA      |
      | ECDSA    |

  @AWSPCAIssuer
  Scenario Outline: Issue a CA certificate with a namespace issuer 
    Given I create an AWSPCAIssuer using a XA CA
    And I update my certificate spec to issue a CA certificate
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | certType |
      | RSA      |
      | ECDSA    |

  @AWSPCAIssuer
  Scenario Outline: Issue a short lived certificate with a namespace issuer 
    Given I create an AWSPCAIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    And I update my certificate spec to issue a certificate with duration of 20 hours
    And I update my certificate spec to issue a certificate with renew before of 5 hours
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | certType |
      | RSA      |
      | ECDSA    |

  @AWSPCAIssuer @KubernetesSecrets
  Scenario Outline: Issue a certificate with a namespace issuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA    |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | myKeyId           | mySecret              | RSA      |
      | myKeyId           | mySecret              | ECDSA    |

  @AWSPCAIssuer @KubernetesSecrets
  Scenario Outline: Issue a CA certificate with a namespace issuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAIssuer using a XA CA
    And I update my certificate spec to issue a CA certificate
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA    |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | myKeyId           | mySecret              | RSA      |
      | myKeyId           | mySecret              | ECDSA    |

  @AWSPCAIssuer @KubernetesSecrets
  Scenario Outline: Issue a short lived certificate with a namespace issuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAIssuer using a XA CA
    And I update my certificate spec to issue a <certType> certificate
    And I update my certificate spec to issue a certificate with duration of 20 hours
    And I update my certificate spec to issue a certificate with renew before of 5 hours
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA    |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType |
      | myKeyId           | mySecret              | RSA      |
      | myKeyId           | mySecret              | ECDSA    |
