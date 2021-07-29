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
        SLACK_MESSAGE = "Data-Node CI » <${RUN_DISPLAY_URL}|Jenkins ${BRANCH_NAME} Job>${ env.CHANGE_URL ? " » <${CHANGE_URL}|GitHub PR #${CHANGE_ID}>" : '' }"
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

        stage('Build docker image') {
            when {
                anyOf {
                    branch 'develop';
                    buildingTag();
                }
            }
            steps {
                retry(3) {
                    dir('data-node') {
                        withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')]) {
                            sh label: 'Log in to a Docker registry', script: '''
                                echo ${PASSWORD} | docker login -u ${USERNAME} --password-stdin docker.pkg.github.com
                            '''
                            sh label: 'Build and push docker image', script: '''
                                mkdir -p docker/bin
                                find cmd -maxdepth 1 -and -not -name cmd | sed -e 's#^cmd/##' | while read -r app ; do
                                    cp -a "cmd/$app/$app-linux-amd64" "docker/bin/$app" || exit 1 ;
                                done
                                tmptag="$(openssl rand -hex 10)"
                                imagetag=$BRANCH_NAME
                                if [[ $BRANCH_NAME == "develop" ]]; then
                                    imagetag=edge
                                fi
                                ls -al docker/bin
                                docker build -t "docker.pkg.github.com/vegaprotocol/data-node/data-node:$tmptag" docker/
                                rm -rf docker/bin
                                docker tag "docker.pkg.github.com/vegaprotocol/data-node/data-node:$tmptag" "docker.pkg.github.com/vegaprotocol/data-node/data-node:$imagetag"
                                docker push "docker.pkg.github.com/vegaprotocol/data-node/data-node:$imagetag"
                                docker rmi "docker.pkg.github.com/vegaprotocol/data-node/data-node:$imagetag"
                            '''
                        }
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

        stage('Build docker image on PR') {
            when {
                changeRequest()
            }
            steps {
                retry(3) {
                    dir('data-node') {
                        withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')]) {
                            sh label: 'Log in to a Docker registry', script: '''
                                echo ${PASSWORD} | docker login -u ${USERNAME} --password-stdin docker.pkg.github.com
                            '''
                            sh label: 'Build docker image', script: '''
                                mkdir -p docker/bin
                                find cmd -maxdepth 1 -and -not -name cmd | sed -e 's#^cmd/##' | while read -r app ; do
                                    cp -a "cmd/$app/$app-linux-amd64" "docker/bin/$app" || exit 1 ;
                                    done
                                tmptag="$(openssl rand -hex 10)"
                                ls -al docker/bin
                                docker build -t "docker.pkg.github.com/vegaprotocol/data-node/data-node:$tmptag" docker/
                                rm -rf docker/bin
                                docker rmi "docker.pkg.github.com/vegaprotocol/data-node/data-node:$tmptag"
                            '''
                        }
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
