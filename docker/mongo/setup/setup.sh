#!/bin/bash

echo ****************************************************************************
echo Starting MongoDB replicaSet
echo ****************************************************************************

sleep 10 | echo Waiting
mongo mongo-primary:27017 replicaSet.js