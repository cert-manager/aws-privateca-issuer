@CrossAccount
Feature: Issue certificates using a CA in another account 
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates from a CA in another account

  Scenario Outline: Issue a certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a <caType> CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | caType | certType       |
      | XA     | SHORT_VALIDITY |
      | XA     | RSA            |
      | XA     | ECDSA          |
      | XA     | CA             |

  Scenario Outline: Issue a certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> AWS_ACCESS_KEY_ID and <secretKeyId> AWS_SECRET_ACCESS_KEY for my AWS credentials
    And I create an AWSPCAClusterIssuer using a <caType> CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    @KubernetesSecrets
    Examples:
      | accessKeyId       | secretKeyId           | caType | certType      |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA    | SHORT_VALIDITY |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA    | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA    | ECDSA          |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA    | CA             |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | caType | certType      |
      | myKeyId           | mySecret              | XA    | SHORT_VALIDITY |
      | myKeyId           | mySecret              | XA    | RSA            |
      | myKeyId           | mySecret              | XA    | ECDSA          |
      | myKeyId           | mySecret              | XA    | CA             |

  Scenario Outline: Issue a certificate with a namespace issuer 
    Given I create an AWSPCAIssuer using a <caType> CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    Examples:
      | caType | certType       |
      | XA     | SHORT_VALIDITY |
      | XA     | RSA            |
      | XA     | ECDSA          |
      | XA     | CA             |

  Scenario Outline: Issue a certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAIssuer using a <caType> CA
    When I issue a <certType> certificate
    Then the certificate should be issued successfully

    @KubernetesSecrets
    Examples:
      | accessKeyId       | secretKeyId           | caType | certType       |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA     | SHORT_VALIDITY |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA     | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA     | ECDSA          |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | XA     | CA             |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | caType | certType       |
      | myKeyId           | mySecret              | XA     | SHORT_VALIDITY |
      | myKeyId           | mySecret              | XA     | RSA            |
      | myKeyId           | mySecret              | XA     | ECDSA          |
      | myKeyId           | mySecret              | XA     | CA             |
      | myKeyId           | mySecret              | ECDSA  | SHORT_VALIDITY |
      | myKeyId           | mySecret              | ECDSA  | RSA            |
      | myKeyId           | mySecret              | ECDSA  | ECDSA          |
      | myKeyId           | mySecret              | ECDSA  | CA             |

