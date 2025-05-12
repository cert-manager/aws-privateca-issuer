@AWSPCAClusterIssuer
Feature: Issue certificates using an AWSPCAClusterIssuer 
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates using an AWSPCAClusterIssuer

  Scenario Outline: Issue a certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a <caType> CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | caType | certType       |
      | RSA    | SHORT_VALIDITY |
      | RSA    | RSA            |
      | RSA    | ECDSA          |
      | RSA    | CA             |
      | ECDSA  | SHORT_VALIDITY |
      | ECDSA  | RSA            |
      | ECDSA  | ECDSA          |
      | ECDSA  | CA             |

  @KubernetesSecrets
  Scenario Outline: Issue a certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAClusterIssuer using a <caType> CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | caType | certType       |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA    | SHORT_VALIDITY |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA    | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA    | ECDSA          |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA    | CA             |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA  | SHORT_VALIDITY |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA  | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA  | ECDSA          |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA  | CA             |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | caType | certType       |
      | myKeyId           | mySecret              | RSA    | SHORT_VALIDITY |
      | myKeyId           | mySecret              | RSA    | RSA            |
      | myKeyId           | mySecret              | RSA    | ECDSA          |
      | myKeyId           | mySecret              | RSA    | CA             |
      | myKeyId           | mySecret              | ECDSA  | SHORT_VALIDITY |
      | myKeyId           | mySecret              | ECDSA  | RSA            |
      | myKeyId           | mySecret              | ECDSA  | ECDSA          |
      | myKeyId           | mySecret              | ECDSA  | CA             |

  @CertificateRecovery
  Scenario: Issue a certificate with a non-existent issuer, is successfully issued after the issuer is created
    Given I create an AWSPCAClusterIssuer using a RSA CA
    And I delete the AWSPCAClusterIssuer
    And I issue a RSA certificate
    And the certificate request has been created
    And the certificate request has reason Pending and status False
    When I create an AWSPCAClusterIssuer using a RSA CA
    Then the certificate should be issued successfully

