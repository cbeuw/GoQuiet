mkdir -p release

read -p "Cleaning $PWD/release directory. Proceed? [y/n]" res
if [ ! "$res" == "y" ]; then
	echo "Abort"
	exit 1
fi

rm -rf ./release/*
pushd ./release

declare -a os=("windows" "linux" "darwin")
declare -a arch=("amd64" "386" "arm")

if [ -z "$v" ]; then
	echo "Version number cannot be null. Run with v=[version] release.sh"
	exit 1
fi

echo "Compiling:"

for o in "${os[@]}"
do
	for a in "${arch[@]}"
	do
			
		if [ $o == "darwin" -a $a == "arm" ]; then
			continue
		fi


		if [ $o == "windows" ]; then
			oext="-win"
		elif [ $o == "darwin" ]; then
			oext="-mac"
		else
			oext="-$o"
		fi

		if [ $a == "amd64" ]; then
			aext="64-"
		elif [ $a == "386" ]; then
			aext="32-"
		else
			aext="$a-"
		fi

		ext="$oext$aext$v"

		if [ $o == "windows" ]; then
			ext="$ext.exe"
		fi

		echo "gq-server$ext"
		GOOS=$o GOARCH=$a go build -ldflags "-X main.version=${v}" -o "gq-server$ext" ../cmd/gq-server
		echo "gq-client$ext"
		GOOS=$o GOARCH=$a go build -ldflags "-X main.version=${v}" -o "gq-client$ext" ../cmd/gq-client
	done
done
