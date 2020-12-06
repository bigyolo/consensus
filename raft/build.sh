#! /bin/bash
echo "begin to build app"
int=0
while(( $int<$1 ))
do  
    nohup ./myraft N$int > N$int.log 2>&1 &
    let "int++"
done
echo "build finished"