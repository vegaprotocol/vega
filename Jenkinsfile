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
        parallelsAlwaysFailFast()
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
                stage('[TODO] diff verification (no changes generated by the CI)') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                echo 'Run diff verification (no changes generated by the CI)'
                            }
                        }
                    }
                }
                stage('[TODO] more linting on multiple file format (sh, py, yaml....)') {
                    steps {
                        retry(3) {
                            dir('data-node') {
                                echo 'Run more linting on multiple file format (sh, py, yaml....)'
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


        stage('Run Checks') {
            parallel {
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
                stage('[TODO] test again with a race flag') {
                    steps {
                        retry(3) {
                            dir('vega') {
                                echo 'Run test again with a race flag'
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
                        DOCKER_IMAGE_URL = "https://github.com/vegaprotocol/vega/pkgs/container/vega%2Fvega"
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
                                    docker run -it --rm "${DOCKER_IMAGE_NAME}" version
                                    docker run -it --rm "${DOCKER_IMAGE_NAME}" --help
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
                                    message: ":docker: Published new `data-node` docker image <${DOCKER_IMAGE_URL}|${DOCKER_IMAGE_TAG}>",
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
                        RELEASE_URL = "https://github.com/vegaprotocol/vega/releases/tag/${TAG_NAME}"
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
                                    message: ":rocket: Published new <${RELEASE_URL}|Release ${TAG_NAME}>",
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
                slackSend(channel: "#tradingcore-notify", color: "good", message: ":white_check_mark: ${SLACK_MESSAGE}")
            }
        }
        failure {
            retry(3) {
                slackSend(channel: "#tradingcore-notify", color: "danger", message: ":red_circle: ${SLACK_MESSAGE}")
            }
        }
    }
}
