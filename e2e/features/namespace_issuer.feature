@AWSPCAIssuer
Feature: Issue certificates using an AWSPCAIssuer 
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates using an AWSPCAIssuer so I can scope down permissions to a single namespace

  Background: Create unique namespace
    Given I create a namespace	

  Scenario Outline: Issue a certificate
    Given I create an AWSPCAIssuer using a <caType> CA
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
  Scenario Outline: Issue a certificate using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAIssuer using a <caType> CA
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

