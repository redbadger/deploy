.PHONEY: default
default:
	docker build -t redbadger/deploy .
	docker push redbadger/deploy
