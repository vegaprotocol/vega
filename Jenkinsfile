/* properties of scmVars (example): 
    - GIT_BRANCH:PR-40-head
    - GIT_COMMIT:05a1c6fbe7d1ff87cfc40a011a63db574edad7e6
    - GIT_PREVIOUS_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_PREVIOUS_SUCCESSFUL_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_URL:https://github.com/vegaprotocol/data-node.git
*/
def scmVars = null
def version = 'UNKNOWN'
def versionHash = 'UNKNOWN'


pipeline {
    agent { label 'general' }
    options {
        skipDefaultCheckout true
    }
    environment {
        GO111MODULE = 'on'
    }

    stages {
        stage('Git clone data-node') {
            steps {
                retry(3) {
                    dir('data-node') {
                        script {
                            scmVars = checkout(scm)
                            versionHash = sh (returnStdout: true, script: "echo \"${scmVars.GIT_COMMIT}\"|cut -b1-8").trim()
                            version = sh (returnStdout: true, script: "git describe --tags 2>/dev/null || echo ${versionHash}").trim()
                        }
                    }
                }
            }
        }

        stage('Compile data-node') {
            environment {
                CGO_ENABLED  = 0
                LDFLAGS      = "-X main.CLIVersion=\"${version}\" -X main.CLIVersionHash=\"${versionHash}\""
            }
            parallel {
                stage('Linux build') {
                    environment {
                        GOOS         = 'linux'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-linux-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node'
                                // quick check
                                sh 'file ${OUTPUT}'
                                sh '${OUTPUT} version'
                            }
                        }
                    }
                }
                stage('MacOS build') {
                    environment {
                        GOOS         = 'darwin'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-darwin-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node'
                                // quick check
                                sh 'file ${OUTPUT}'
                            }
                        }
                    }
                }
                stage('Windows build') {
                    environment {
                        GOOS         = 'windows'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-windows-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node'
                                // quick check
                                sh 'file ${OUTPUT}'
                            }
                        }
                    }
                }
            }
        }

        stage('Run checks') {
            parallel {
                stage('[TODO] markdown verification') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run markdown verification'
                            }
                        }
                    }
                }
                stage('[TODO] unit tests') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run unit tests'
                            }
                        }
                    }
                }
                stage('[TODO] integration tests') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run integration tests'
                            }
                        }
                    }
                }
                stage('[TODO] check gqlgen') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run check gqlgen'
                            }
                        }
                    }
                }
                stage('[TODO] check print') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run check print'
                            }
                        }
                    }
                }
                stage('[TODO] check proto') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run check proto'
                            }
                        }
                    }
                }
                stage('[TODO] test again with a race flag') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run test again with a race flag'
                            }
                        }
                    }
                }
                stage('[TODO] vet') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run vet'
                            }
                        }
                    }
                }
                stage('[TODO] code owner') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run code owner'
                            }
                        }
                    }
                }
                stage('[TODO] buf lint') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run buf lint'
                            }
                        }
                    }
                }
                stage('[TODO] misspell') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run misspell'
                            }
                        }
                    }
                }
                stage('[TODO] static check') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run static check'
                            }
                        }
                    }
                }
                stage('[TODO] swagger diff verification') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run swagger diff verification'
                            }
                        }
                    }
                }
                stage('[TODO] diff verification (no changes generated by the CI)') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run diff verification (no changes generated by the CI)'
                            }
                        }
                    }
                }
                stage('[TODO] more linting on multiple file format (sh, py, yaml....)') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run more linting on multiple file format (sh, py, yaml....)'
                            }
                        }
                    }
                }
                stage('[TODO] feature (integration) tests from specs-internal repo') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run feature (integration) tests from specs-internal repo'
                            }
                        }
                    }
                }
                stage('[TODO] system-tests') {
                    steps {
                        dir('system-tests') {
                            echo 'Run system-tests'
                        }
                    }
                }
            }
        }

        stage('[TODO] Deploy to Devnet') {
            when {
                branch 'develop'
            }
            steps {
                echo 'Deploying to Devnet....'
            }
        }

        stage('[TODO] Basic tests Devnet') {
            when {
                branch 'develop'
            }
            steps {
                echo 'Run basic tests on Devnet network ...'
            }
        }

        stage('[TODO] Do something on master') {
            when {
                branch 'master'
            }
            steps {
                echo 'Do something on master....'
            }
        }

        stage('[TODO] Build and publish version') {
            when { tag "v*" }
            steps {
                echo 'Build version because this commit is tagged...'
                echo 'and publish it'
            }
        }
    }
}
