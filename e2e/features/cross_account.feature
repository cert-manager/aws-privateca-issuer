@CrossAccount
Feature: Issue certificates using a CA in another account 
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates from a CA in another account

  @AWSPCAClusterIssuer
  Scenario Outline: Issue a certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a XA CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | certType       |
      | SHORT_VALIDITY |
      | RSA            |
      | ECDSA          |
      | CA             |

  @AWSPCAClusterIssuer @KubernetesSecrets   
  Scenario Outline: Issue a certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> AWS_ACCESS_KEY_ID and <secretKeyId> AWS_SECRET_ACCESS_KEY for my AWS credentials
    And I create an AWSPCAClusterIssuer using a XA CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | SHORT_VALIDITY |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA          |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | CA             |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType      |
      | myKeyId           | mySecret              | SHORT_VALIDITY |
      | myKeyId           | mySecret              | RSA            |
      | myKeyId           | mySecret              | ECDSA          |
      | myKeyId           | mySecret              | CA             |

  @AWSPCAIssuer
  Scenario Outline: Issue a certificate with a namespace issuer 
    Given I create an AWSPCAIssuer using a XA CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | certType       |
      | SHORT_VALIDITY |
      | RSA            |
      | ECDSA          |
      | CA             |

  @AWSPCAIssuer @KubernetesSecrets
  Scenario Outline: Issue a certificate with a namespace issuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAIssuer using a XA CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | certType       |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | SHORT_VALIDITY |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA          |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | CA             |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | certType       |
      | myKeyId           | mySecret              | SHORT_VALIDITY |
      | myKeyId           | mySecret              | RSA            |
      | myKeyId           | mySecret              | ECDSA          |
      | myKeyId           | mySecret              | CA             |

