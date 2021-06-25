pipeline {
    agent any
    options {
        skipDefaultCheckout true
    }
    parameters {
        string(name: 'VEGA_CORE_BRANCH', defaultValue: "${CHANGE_BRANCH}", description: 'Git branch name of the vegaprotocol/vega repository')
        string(name: 'SYSTEM_TESTS_BRANCH', defaultValue: 'develop', description: 'Git branch name of the vegaprotocol/system-tests repository')
        string(name: 'DEVOPS_INFRA_BRANCH', defaultValue: 'master', description: 'Git branch name of the vegaprotocol/devops-infra repository')
    }

    stages {
        stage('setup') {
            steps {
                sh 'printenv'
                echo "${params}"
            }
        }
        stage('Git clone') {
            parallel {
                stage('vega core') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                checkout scm
                            }
                        }
                    }
                }
                stage('system-tests') {
                    steps {
                        retry(3) {
                            dir('system-tests') {
                                git branch: "${params.SYSTEM_TESTS_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/system-tests.git'
                            }
                        }
                    }
                }
                stage('devops-infra') {
                    steps {
                        retry(3) {
                            dir('devops-infra') {
                                git branch: "${params.DEVOPS_INFRA_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/devops-infra.git'
                            }
                        }
                    }
                }
            }
        }

        stage('Compile vega core') {
            parallel {
                stage('for Linux') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Compile Vega Core for Linux'
                            }
                        }
                    }
                }
                stage('for MacOS') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Compile Vega Core for MacOS'
                            }
                        }
                    }
                }
                stage('for Windows') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Compile Vega Core for Windows'
                            }
                        }
                    }
                }
            }
        }

        stage('Run checks') {
            parallel {
                stage('markdown verification') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run markdown verification'
                            }
                        }
                    }
                }
                stage('unit tests') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run unit tests'
                            }
                        }
                    }
                }
                stage('integration tests') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run integration tests'
                            }
                        }
                    }
                }
                stage('check gqlgen') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run check gqlgen'
                            }
                        }
                    }
                }
                stage('check print') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run check print'
                            }
                        }
                    }
                }
                stage('check proto') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run check proto'
                            }
                        }
                    }
                }
                stage('test again with a race flag') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run test again with a race flag'
                            }
                        }
                    }
                }
                stage('vet') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run vet'
                            }
                        }
                    }
                }
                stage('code owner') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run code owner'
                            }
                        }
                    }
                }
                stage('buf lint') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run buf lint'
                            }
                        }
                    }
                }
                stage('misspell') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run misspell'
                            }
                        }
                    }
                }
                stage('static check') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run static check'
                            }
                        }
                    }
                }
                stage('swagger diff verification') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run swagger diff verification'
                            }
                        }
                    }
                }
                stage('diff verification (no changes generated by the CI)') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run diff verification (no changes generated by the CI)'
                            }
                        }
                    }
                }
                stage('more linting on multiple file format (sh, py, yaml....)') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run more linting on multiple file format (sh, py, yaml....)'
                            }
                        }
                    }
                }
                stage('feature (integration) tests from specs-internal repo') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run feature (integration) tests from specs-internal repo'
                            }
                        }
                    }
                }
                stage('system-tests') {
                    steps {
                        dir('system-tests') {
                            echo 'Run system-tests'
                        }
                    }
                }
            }
        }

        stage('Deploy to Devnet') {
            when {
                branch 'develop'
            }
            steps {
                echo 'Deploying to Devnet....'
            }
        }

        stage('Basic tests Devnet') {
            when {
                branch 'develop'
            }
            steps {
                echo 'Run basic tests on Devnet network ...'
            }
        }
    }
}
