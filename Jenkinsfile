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
        CGO_ENABLED  = 0
        SLACK_MESSAGE = "Data-Node CI » <${RUN_DISPLAY_URL}|Jenkins ${BRANCH_NAME} Job>${ env.CHANGE_URL ? " » <${CHANGE_URL}|GitHub PR #${CHANGE_ID}>" : '' }"
    }

    stages {
        stage('Git Clone') {
            steps {
                sh 'printenv'
                echo "${params}"
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

        stage('go mod download deps') {
            steps {
                retry(3) {
                    dir('data-node') {
                        sh 'go mod download -x'
                    }
                }
            }
        }

        stage('Build') {
            environment {
                LDFLAGS      = "-X main.CLIVersion=\"${version}\" -X main.CLIVersionHash=\"${versionHash}\""
            }
            failFast true
            parallel {
                stage('Linux') {
                    environment {
                        GOOS         = 'linux'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-linux-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh label: 'Compile', script: '''
                                    go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node
                                '''
                                sh label: 'Sanity check', script: '''
                                    file ${OUTPUT}
                                    ${OUTPUT} version
                                '''
                            }
                        }
                    }
                }
                stage('MacOS') {
                    environment {
                        GOOS         = 'darwin'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-darwin-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh label: 'Compile', script: '''
                                    go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node
                                '''
                                sh label: 'Sanity check', script: '''
                                    file ${OUTPUT}
                                '''
                            }
                        }
                    }
                }
                stage('Windows') {
                    environment {
                        GOOS         = 'windows'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-windows-amd64'
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh label: 'Compile', script: '''
                                    go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node
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

        stage('Run linters') {
            parallel {
                stage('static check') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'staticcheck -checks "all,-SA1019,-ST1000,-ST1021" ./...'
                            }
                        }
                    }
                }
                stage('go vet') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'go vet ./...'
                            }
                        }
                    }
                }
                stage('check print') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'make print_check'
                            }
                        }
                    }
                }
                stage('misspell') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'golangci-lint run --disable-all --enable misspell'
                            }
                        }
                    }
                }
                stage('shellcheck') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh "git ls-files '*.sh'"
                                sh "git ls-files '*.sh' | xargs shellcheck"
                            }
                        }
                    }
                }
                stage('yamllint') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh "git ls-files '*.yml' '*.yaml'"
                                sh "git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 160}}}'"
                            }
                        }
                    }
                }
                stage('python files') {
                    steps {
                        retry(3) {
                            dir('data-node') {
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
                            dir('data-node') {
                                sh "git ls-files '*.json'"
                                sh "for f in \$(git ls-files '*.json'); do echo \"check \$f\"; jq empty \"\$f\"; done"
                            }
                        }
                    }
                }
                stage('markdown spellcheck') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md"'
                            }
                        }
                    }
                }
            }
        }

        stage('Run tests') {
            parallel {
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
                        DOCKER_IMAGE_TAG = "${ env.TAG_NAME ? env.TAG_NAME : env.BRANCH_NAME }"
                        DOCKER_IMAGE_NAME = "docker.pkg.github.com/vegaprotocol/data-node/data-node:${DOCKER_IMAGE_TAG}"
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                sh label: 'Build docker image', script: '''#!/bin/bash -e
                                    mkdir -p docker/bin
                                    cp -a "cmd/data-node/data-node-linux-amd64" "docker/bin/data-node"
                                    docker build -t "${DOCKER_IMAGE_NAME}" docker/
                                    rm -rf docker/bin
                                '''
                                sh label: 'Sanity check', script: '''
                                    docker run --rm "${DOCKER_IMAGE_NAME}" version
                                '''
                                withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')]) {
                                    sh label: 'Log in to a Docker registry', script: '''
                                        echo ${PASSWORD} | docker login -u ${USERNAME} --password-stdin docker.pkg.github.com
                                    '''
                                }
                                sh label: 'Push docker image', script: '''#!/bin/bash -e
                                    docker push "${DOCKER_IMAGE_NAME}"
                                    docker rmi "${DOCKER_IMAGE_NAME}"
                                '''
                                slackSend(
                                    channel: "#tradingcore-notify",
                                    color: "good",
                                    message: ":docker: Data-Node » Published new docker image `${DOCKER_IMAGE_NAME}`",
                                )
                            }
                        }
                    }
                    post {
                        always  {
                            retry(3) {
                                script {
                                    sh label: 'Log out from the Docker registry', script: '''
                                        docker logout docker.pkg.github.com
                                    '''
                                }
                            }
                        }
                    }
                }

                stage('release to GitHub') {
                    when {
                        buildingTag()
                    }
                    environment {
                        RELEASE_URL = "https://github.com/vegaprotocol/data-node/releases/tag/${TAG_NAME}"
                    }
                    steps {
                        retry(3) {
                            dir('data-node') {
                                withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', passwordVariable: 'TOKEN', usernameVariable:'USER')]) {
                                    // Workaround for user input:
                                    //  - global configuration: 'gh config set prompt disabled'
                                    sh label: 'Log in to a Gihub with CI', script: '''
                                        echo ${TOKEN} | gh auth login --with-token -h github.com
                                    '''
                                }
                                sh label: 'Upload artifacts', script: '''#!/bin/bash -e
                                    [[ $TAG_NAME =~ '-pre' ]] && prerelease='--prerelease' || prerelease=''
                                    gh release create $TAG_NAME $prerelease ./cmd/data-node/data-node-*
                                '''
                                slackSend(
                                    channel: "#tradingcore-notify",
                                    color: "good",
                                    message: ":rocket: Data-Node » Published new version to GitHub <${RELEASE_URL}|${TAG_NAME}>",
                                )
                            }
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
    }
}
