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
        timestamps()
    }
    parameters {
        string(name: 'SYSTEM_TESTS_BRANCH', defaultValue: 'develop', description: 'Git branch name of the vegaprotocol/system-tests repository')
        string(name: 'DEVOPS_INFRA_BRANCH', defaultValue: 'master', description: 'Git branch name of the vegaprotocol/devops-infra repository')
        string(name: 'SPECS_INTERNAL_BRANCH', defaultValue: 'master', description: 'Git branch name of the vegaprotocol/specs-internal repository')
        string(name: 'PROTOS_BRANCH', defaultValue: 'develop', description: 'Git branch name of the vegaprotocol/protos repository')
    }
    environment {
        CGO_ENABLED = 0
        GO111MODULE = 'on'
        SLACK_MESSAGE = "Vega Core CI » <${RUN_DISPLAY_URL}|Jenkins ${BRANCH_NAME} Job>${ env.CHANGE_URL ? " » <${CHANGE_URL}|GitHub PR #${CHANGE_ID}>" : '' }"
        LOCAL_DOCKER_IMAGE_NAME = "docker.pkg.github.com/vegaprotocol/vega/vega:${BRANCH_NAME}"
    }

    stages {
        stage('Git clone') {
            parallel {
                stage('vega core') {
                    options { retry(3) }
                    steps {
                        sh 'printenv'
                        echo "${params}"
                        dir('vega') {
                            script {
                                scmVars = checkout(scm)
                                versionHash = sh (returnStdout: true, script: "echo \"${scmVars.GIT_COMMIT}\"|cut -b1-8").trim()
                                version = sh (returnStdout: true, script: "git describe --tags 2>/dev/null || echo ${versionHash}").trim()
                            }
                        }
                    }
                }
                stage('specs-internal') {
                    options { retry(3) }
                    steps {
                        dir('specs-internal') {
                            git branch: "${params.SPECS_INTERNAL_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/specs-internal.git'
                        }
                    }
                }
                stage('protos') {
                    options { retry(3) }
                    steps {
                        dir('protos') {
                            git branch: "${params.PROTOS_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/protos.git'
                        }
                    }
                }
                stage('system-tests') {
                    options { retry(3) }
                    steps {
                        dir('system-tests') {
                            git branch: "${params.SYSTEM_TESTS_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/system-tests.git'
                        }
                    }
                }
                stage('devops-infra') {
                    options { retry(3) }
                    steps {
                        dir('devops-infra') {
                            git branch: "${params.DEVOPS_INFRA_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/devops-infra.git'
                        }
                    }
                }
            }
        }

        stage('go mod download deps') {
            options { retry(3) }
            steps {
                dir('vega') {
                    sh 'go mod download -x'
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
                    options { retry(3) }
                    steps {
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
                stage('MacOS build') {
                    environment {
                        GOOS         = 'darwin'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-darwin-amd64'
                    }
                    options { retry(3) }
                    steps {
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
                stage('Windows build') {
                    environment {
                        GOOS         = 'windows'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-windows-amd64'
                    }
                    options { retry(3) }
                    steps {
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

        stage(' ') {
            failFast true
            parallel {
                // this task needs to run after builds
                stage('Build docker image') {
                    environment {
                        LINUX_BINARY = './cmd/vega/vega-linux-amd64'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'Copy binary', script: '''#!/bin/bash -e
                                mkdir -p docker/bin
                                cp -a "${LINUX_BINARY}" "docker/bin/vega"
                            '''
                            // Note: This docker image is used by system-tests and publish stage
                            withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://docker.pkg.github.com"]) {
                                sh label: 'Build docker image', script: '''
                                    docker build -t "${LOCAL_DOCKER_IMAGE_NAME}" docker/
                                '''
                            }
                            sh label: 'Cleanup', script: '''#!/bin/bash -e
                                rm -rf docker/bin
                            '''
                            sh label: 'Sanity check', script: '''
                                docker run --rm --entrypoint "" "${LOCAL_DOCKER_IMAGE_NAME}" vega version
                            '''
                        }
                    }
                }
                // this task needs to run before linters and tests
                stage('Run gqlgen codgen checks') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'make gqlgen_check'
                        }
                    }
                }
            }
        }

        stage('Run linters') {
            parallel {
                stage('buf lint') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'buf lint'
                        }
                    }
                }
                stage('static check') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'staticcheck -checks "all,-SA1019,-ST1000,-ST1021" ./...'
                        }
                    }
                }
                stage('go vet') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go vet ./...'
                        }
                    }
                }
                stage('check print') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'make print_check'
                        }
                    }
                }
                stage('misspell') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'golangci-lint run --disable-all --enable misspell'
                        }
                    }
                }
                stage('shellcheck') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh "git ls-files '*.sh'"
                            sh "git ls-files '*.sh' | xargs shellcheck"
                        }
                    }
                }
                stage('yamllint') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh "git ls-files '*.yml' '*.yaml'"
                            sh "git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 160}}}'"
                        }
                    }
                }
                stage('json format') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh "git ls-files '*.json'"
                            sh "for f in \$(git ls-files '*.json'); do echo \"check \$f\"; jq empty \"\$f\"; done"
                        }
                    }
                }
                stage('markdown spellcheck') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md"'
                        }
                    }
                }
            }
        }

        stage('Run tests') {
            parallel {
                stage('unit tests with race') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go test -v -race ./... 2>&1 | tee unit-test-race-results.txt && cat unit-test-race-results.txt | go-junit-report > vega-unit-test-race-report.xml'
                            junit checksName: 'Unit Tests with Race', testResults: 'vega-unit-test-race-report.xml'
                        }
                    }
                }
                stage('unit tests') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go test -v ./... 2>&1 | tee unit-test-results.txt && cat unit-test-results.txt | go-junit-report > vega-unit-test-report.xml'
                            junit checksName: 'Unit Tests', testResults: 'vega-unit-test-report.xml'
                        }
                    }
                }
                stage('vega/integration tests') {
                    options { retry(3) }
                    steps {
                        dir('vega/integration') {
                            sh 'godog build -o integration.test && ./integration.test --format=junit:vega-integration-report.xml'
                            junit checksName: 'Integration Tests', testResults: 'vega-integration-report.xml'
                        }
                    }
                }
                stage('specs-internal qa-scenarios') {
                    options { retry(3) }
                    steps {
                        dir('vega/integration') {
                            sh 'godog build -o qa_integration.test && ./qa_integration.test --format=junit:specs-internal-qa-scenarios-report.xml ../../specs-internal/qa-scenarios/'
                            junit checksName: 'Specs Tests (specs-internal)', testResults: 'specs-internal-qa-scenarios-report.xml'
                        }
                    }
                }
                stage('system-tests') {
                    stages {
                        stage('check') {
                            steps {
                                dir('system-tests/scripts') {
                                    sh label: 'Check setup', script: '''
                                        make check
                                    '''
                                }
                            }
                        }
                        stage('docker pull') {
                            steps {
                                dir('system-tests/scripts') {
                                    withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://docker.pkg.github.com"]) {
                                        sh 'make prepare-docker-pull'
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }


        stage('Publish') {
            parallel {

                stage('docker image') {
                    when {
                        anyOf {
                            buildingTag()
                            branch 'develop'
                            // changeRequest() // uncomment only for testing
                        }
                    }
                    environment {
                        DOCKER_IMAGE_TAG_VERSIONED = "${ env.TAG_NAME ? env.TAG_NAME : env.BRANCH_NAME }"
                        DOCKER_IMAGE_NAME_VERSIONED = "docker.pkg.github.com/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_VERSIONED}"
                        DOCKER_IMAGE_TAG_ALIAS = "${ env.TAG_NAME ? 'latest' : 'edge' }"
                        DOCKER_IMAGE_NAME_ALIAS = "docker.pkg.github.com/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_ALIAS}"
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'Tag new images', script: '''#!/bin/bash -e
                                docker image tag "${LOCAL_DOCKER_IMAGE_NAME}" "${DOCKER_IMAGE_NAME_VERSIONED}"
                                docker image tag "${LOCAL_DOCKER_IMAGE_NAME}" "${DOCKER_IMAGE_NAME_ALIAS}"
                            '''

                            withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://docker.pkg.github.com"]) {
                                sh label: 'Push docker images', script: '''
                                    docker push "${DOCKER_IMAGE_NAME_VERSIONED}"
                                    docker push "${DOCKER_IMAGE_NAME_ALIAS}"
                                '''
                            }
                            slackSend(
                                channel: "#tradingcore-notify",
                                color: "good",
                                message: ":docker: Vega Core » Published new docker image `${DOCKER_IMAGE_NAME_VERSIONED}` aka `${DOCKER_IMAGE_NAME_ALIAS}`",
                            )
                        }
                    }
                }

                stage('release to GitHub') {
                    when {
                        buildingTag()
                    }
                    environment {
                        RELEASE_URL = "https://github.com/vegaprotocol/vega/releases/tag/${TAG_NAME}"
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', passwordVariable: 'TOKEN', usernameVariable:'USER')]) {
                                // Workaround for user input:
                                //  - global configuration: 'gh config set prompt disabled'
                                sh label: 'Log in to a Gihub with CI', script: '''
                                    echo ${TOKEN} | gh auth login --with-token -h github.com
                                '''
                            }
                            sh label: 'Upload artifacts', script: '''#!/bin/bash -e
                                [[ $TAG_NAME =~ '-pre' ]] && prerelease='--prerelease' || prerelease=''
                                gh release create $TAG_NAME $prerelease ./cmd/vega/vega-*
                            '''
                            slackSend(
                                channel: "#tradingcore-notify",
                                color: "good",
                                message: ":rocket: Vega Core » Published new version to GitHub <${RELEASE_URL}|${TAG_NAME}>",
                            )
                        }
                    }
                    post {
                        always  {
                            retry(3) {
                                script {
                                    sh label: 'Log out from Github', script: '''
                                        gh auth logout -h github.com
                                    '''
                                }
                            }
                        }
                    }
                }

                stage('[TODO] deploy to Devnet') {
                    when {
                        branch 'develop'
                    }
                    options { retry(3) }
                    steps {
                        echo 'Deploying to Devnet....'
                        echo 'Run basic tests on Devnet network ...'
                    }
                }
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
        always {
            retry(3) {
                sh label: 'Clean docker images', script: '''
                    docker rmi "${LOCAL_DOCKER_IMAGE_NAME}"
                '''
            }
        }
    }
}
