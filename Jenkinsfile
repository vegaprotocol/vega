/* properties of scmVars (example):
    - GIT_BRANCH:PR-40-head
    - GIT_COMMIT:05a1c6fbe7d1ff87cfc40a011a63db574edad7e6
    - GIT_PREVIOUS_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_PREVIOUS_SUCCESSFUL_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_URL:https://github.com/vegaprotocol/vega.git
*/
def scmVars = null
def version = 'UNKNOWN'
def versionHash = 'UNKNOWN'


pipeline {
    agent { label 'general' }
    options {
        skipDefaultCheckout true
    }
    parameters {
        string(name: 'SYSTEM_TESTS_BRANCH', defaultValue: 'develop', description: 'Git branch name of the vegaprotocol/system-tests repository')
        string(name: 'DEVOPS_INFRA_BRANCH', defaultValue: 'master', description: 'Git branch name of the vegaprotocol/devops-infra repository')
    }
    environment {
        CGO_ENABLED = 0
        GO111MODULE = 'on'
        SLACK_MESSAGE = "Vega Core CI » <${RUN_DISPLAY_URL}|Jenkins ${BRANCH_NAME} Job>${ env.CHANGE_URL ? " » <${CHANGE_URL}|GitHub PR #${CHANGE_ID}>" : '' }"
    }

    stages {
        stage('Git clone') {
            parallel {
                stage('vega core') {
                    steps {
                        sh 'printenv'
                        echo "${params}"
                        retry(3) {
                            dir('vega') {
                                script {
                                    scmVars = checkout(scm)
                                    versionHash = sh (returnStdout: true, script: "echo \"${scmVars.GIT_COMMIT}\"|cut -b1-8").trim()
                                    version = sh (returnStdout: true, script: "git describe --tags 2>/dev/null || echo ${versionHash}").trim()
                                }
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
                stage('specs-internal') {
                    steps {
                        retry(3) {
                            dir('specs-internal') {
                                git branch: 'master', credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/specs-internal.git'
                            }
                        }
                    }
                }
            }
        }

        stage('go mod download deps') {
            steps {
                retry(3) {
                    dir('vega') {
                        sh 'go mod download -x'
                    }
                }
            }
        }

        stage('Compile vega core') {
            environment {
                LDFLAGS      = "-X main.CLIVersion=\"${version}\" -X main.CLIVersionHash=\"${versionHash}\""
            }
            failFast true
            parallel {
                stage('Linux build') {
                    environment {
                        GOOS         = 'linux'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-linux-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh label: 'Compile', script: '''
                                    go build -v -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/vega
                                '''
                                sh label: 'Sanity check', script: '''
                                    file ${OUTPUT}
                                    ${OUTPUT} version
                                '''
                            }
                        }
                    }
                }
                stage('MacOS build') {
                    environment {
                        GOOS         = 'darwin'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-darwin-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh label: 'Compile', script: '''
                                    go build -v -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/vega
                                '''
                                sh label: 'Sanity check', script: '''
                                    file ${OUTPUT}
                                '''
                            }
                        }
                    }
                }
                stage('Windows build') {
                    environment {
                        GOOS         = 'windows'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-windows-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh label: 'Compile', script: '''
                                    go build -v -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/vega
                                '''
                                sh label: 'Sanity check', script: '''
                                    file ${OUTPUT}
                                '''
                            }
                        }
                    }
                }
            }
        }

        // these stages are run in sequence as they delete and recreate files
        stage('Run gqlgen codgen checks') {
            steps {
                retry(3) {
                    dir('vega') {
                        sh 'make gqlgen_check'
                    }
                }
            }
        }

        stage('Run linters') {
            parallel {
                stage('buf lint') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'buf lint'
                            }
                        }
                    }
                }
                stage('static check') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'staticcheck -checks "all,-SA1019,-ST1000,-ST1021" ./...'
                            }
                        }
                    }
                }
                stage('go vet') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'go vet ./...'
                            }
                        }
                    }
                }
                stage('check print') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'make print_check'
                            }
                        }
                    }
                }
                stage('misspell') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'golangci-lint run --disable-all --enable misspell'
                            }
                        }
                    }
                }
                stage('shellcheck') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh "git ls-files '*.sh'"
                                sh "git ls-files '*.sh' | xargs shellcheck"
                            }
                        }
                    }
                }
                stage('yamllint') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh "git ls-files '*.yml' '*.yaml'"
                                sh "git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 160}}}'"
                            }
                        }
                    }
                }
                stage('python files') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh "git ls-files '*.py'"
                                sh "git ls-files '*.py' | xargs flake8"
                                sh "git ls-files '*.py' | xargs black -l 79 --check --diff"
                            }
                        }
                    }
                }
                stage('json format') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh "git ls-files '*.json'"
                                sh "for f in \$(git ls-files '*.json'); do echo \"check \$f\"; jq empty \"\$f\"; done"
                            }
                        }
                    }
                }
                stage('markdown spellcheck') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md"'
                            }
                        }
                    }
                }
            }
        }

        stage('Run tests') {
            parallel {
                stage('unit tests with race') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'go test -v -race ./... 2>&1 | tee unit-test-race-results.txt && cat unit-test-race-results.txt | go-junit-report > vega-unit-test-race-report.xml'
                                junit checksName: 'Unit Tests with Race', testResults: 'vega-unit-test-race-report.xml'
                            }
                        }
                    }
                }
                stage('unit tests') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                sh 'go test -v ./... 2>&1 | tee unit-test-results.txt && cat unit-test-results.txt | go-junit-report > vega-unit-test-report.xml'
                                junit checksName: 'Unit Tests', testResults: 'vega-unit-test-report.xml'
                            }
                        }
                    }
                }
                stage('vega/integration tests') {
                    steps {
                        retry(3) {
                            dir('vega/integration') {
                                sh 'godog build -o integration.test && ./integration.test --format=junit:vega-integration-report.xml'
                                junit checksName: 'Integration Tests', testResults: 'vega-integration-report.xml'
                            }
                        }
                    }
                }
                stage('specs-internal qa-scenarios') {
                    steps {
                        retry(3) {
                            dir('vega/integration') {
                                sh 'godog build -o qa_integration.test && ./qa_integration.test --format=junit:specs-internal-qa-scenarios-report.xml ../../specs-internal/qa-scenarios/'
                                junit checksName: 'Specs Tests (specs-internal)', testResults: 'specs-internal-qa-scenarios-report.xml'
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
    post {
        success {
            retry(3) {
                slackSend(channel: "#tradingcore-notify", color: "good", message: ":white_check_mark: ${SLACK_MESSAGE} (${currentBuild.durationString.minus(' and counting')})")
            }
        }
        failure {
            retry(3) {
                slackSend(channel: "#tradingcore-notify", color: "danger", message: ":red_circle: ${SLACK_MESSAGE} (${currentBuild.durationString.minus(' and counting')})")
            }
        }
    }
}
