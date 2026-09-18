pipeline {
  agent any

  environment {
    IMAGE_NAME = 'user-service'
    DOCKER_CREDENTIALS = credentials('docker-credential')
    GITHUB_CREDENTIALS = credentials('github-credential')
    CONSUL_HTTP_URL = credentials('consul-http-url')

    // Must match the CONSUL_HTTP_PATH variable read by config.Init() (config/config.go)
    // and declared in .env.example - previously named CONSUL_HTTP_KEY here, which never
    // matched what the app actually reads.
    CONSUL_HTTP_PATH = "backend/user-service"
    CONSUL_HTTP_TOKEN = credentials('consul-http-token')
    CONSUL_WATCH_INTERVAL_SECONDS = credentials('consul-watch-interval-seconds')
  }

  stages {
    stage('Check Commit Message') {
      steps {
        script {
          def commitMessage = sh(
            script: "git log -1 --pretty=%B",
            returnStdout: true
          ).trim()

          echo "Commit Message: ${commitMessage}"
          if (commitMessage.contains("[skip ci]")) {
            echo "Skipping pipeline due to [skip ci] tag in commit message."
            currentBuild.result = 'ABORTED'
            currentBuild.delete()
            return
          }

          echo "Pipeline will continue. No [skip ci] tag found in commit message."
        }
      }
    }

    stage('Set Target Branch') {
      steps {
        script {
          echo "GIT_BRANCH: ${env.GIT_BRANCH}"
          if (env.GIT_BRANCH == 'origin/main') {
            env.TARGET_BRANCH = 'main'
          } else if (env.GIT_BRANCH == 'origin/development') {
            env.TARGET_BRANCH = 'development'
          }

          echo "TARGET_BRANCH: ${env.TARGET_BRANCH}"
        }
      }
    }

    stage('Checkout Code') {
      steps {
        script {
          def repoUrl = 'https://github.com/rals-dev/rs-user-service.git'

          checkout([$class: 'GitSCM',
            branches: [
              [name: "*/${env.TARGET_BRANCH}"]
            ],
            userRemoteConfigs: [
              [url: repoUrl, credentialsId: 'github-credential']
            ]
          ])

          sh 'ls -lah'
        }
      }
    }

    stage('Login to Docker Hub') {
      steps {
        script {
          withCredentials([usernamePassword(credentialsId: 'docker-credential', passwordVariable: 'DOCKER_PASSWORD', usernameVariable: 'DOCKER_USERNAME')]) {
            sh """
            echo $DOCKER_PASSWORD | docker login -u $DOCKER_USERNAME --password-stdin
            """
          }
        }
      }
    }

    stage('Build and Push Docker Image') {
      steps {
        script {
          def runNumber = currentBuild.number
          sh "docker build -t ${DOCKER_CREDENTIALS_USR}/${IMAGE_NAME}:${runNumber} ."
          sh "docker push ${DOCKER_CREDENTIALS_USR}/${IMAGE_NAME}:${runNumber}"
        }
      }
    }

    stage('Update docker-compose.yml') {
      steps {
        script {
          def runNumber = currentBuild.number
          sh "sed -i 's|image: ${DOCKER_CREDENTIALS_USR}/${IMAGE_NAME}:[0-9]\\+|image: ${DOCKER_CREDENTIALS_USR}/${IMAGE_NAME}:${runNumber}|' docker-compose.yml"
        }
      }
    }

    stage('Commit and Push Changes') {
      steps {
        script {
          sh """
          git config --global user.name 'Jenkins CI'
          git config --global user.email 'jenkins@company.com'
          git remote set-url origin https://${GITHUB_CREDENTIALS_USR}:${GITHUB_CREDENTIALS_PSW}@github.com/rals-dev/rs-user-service.git
          git add docker-compose.yml
          git commit -m 'Update image version to ${TARGET_BRANCH}-${currentBuild.number} [skip ci]' || echo 'No changes to commit'
          git pull origin ${TARGET_BRANCH} --rebase
          git push origin HEAD:${TARGET_BRANCH}
          """
        }
      }
    }

    stage('Deploy to Remote Server') {
      steps {
        script {
          def targetDir = "/home/ralsaditya/rals-soccer/user-service"
          // Scope the SSH key / host / username credentials to this stage only,
          // instead of exposing them as pipeline-wide environment variables.
          // sshUserPrivateKey materializes a proper temp key file with correct
          // permissions and cleans it up afterwards.
          withCredentials([
            sshUserPrivateKey(credentialsId: 'ssh-key', keyFileVariable: 'SSH_KEY_FILE'),
            string(credentialsId: 'host', variable: 'DEPLOY_HOST'),
            string(credentialsId: 'username', variable: 'DEPLOY_USERNAME')
          ]) {
            def sshCommandToServer = """
            ssh -o StrictHostKeyChecking=no -i "${SSH_KEY_FILE}" ${DEPLOY_USERNAME}@${DEPLOY_HOST} '
              if [ -d "${targetDir}/.git" ]; then
                  echo "Directory exists. Pulling latest changes."
                  cd "${targetDir}"
                  git pull origin "${TARGET_BRANCH}"
              else
                  echo "Directory does not exist. Cloning repository."
                  git clone -b "${TARGET_BRANCH}" git@github.com:rals-dev/rs-user-service.git "${targetDir}"
                  cd "${targetDir}"
              fi

              cp .env.example .env
              sed -i "s|^TIMEZONE=.*|TIMEZONE=Asia/Jakarta|" "${targetDir}/.env"
              sed -i "s|^CONSUL_HTTP_URL=.*|CONSUL_HTTP_URL=${CONSUL_HTTP_URL}|" "${targetDir}/.env"
              sed -i "s|^CONSUL_HTTP_PATH=.*|CONSUL_HTTP_PATH=${CONSUL_HTTP_PATH}|" "${targetDir}/.env"
              sed -i "s|^CONSUL_HTTP_TOKEN=.*|CONSUL_HTTP_TOKEN=${CONSUL_HTTP_TOKEN}|" "${targetDir}/.env"
              sed -i "s|^CONSUL_WATCH_INTERVAL_SECONDS=.*|CONSUL_WATCH_INTERVAL_SECONDS=${CONSUL_WATCH_INTERVAL_SECONDS}|" "${targetDir}/.env"
              sudo docker compose up -d --build --force-recreate
            '
            """
            sh sshCommandToServer
          }
        }
      }
    }
  }
}
