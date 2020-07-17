#!/usr/bin/env python3

import os
import random
import time

def send_metric(name, value, tp):
    os.system(f'echo "{name}:{value}|{tp}" | nc -u -w0 127.0.0.1 8125')

def rndstring(size = 10):
    chars = "qwertyuiopasdfghjklzxcvbnm"
    s = [ chars[random.randrange(len(chars))] for i in range(size) ]
    return "".join(s)

TYPES = [ 'c', 'g', 'h', 'm' ]
def rndtype():
    return TYPES[random.randrange(len(TYPES))]

def rndsleep():
    time.sleep(random.randrange(1000) / 1000.0)

if __name__ == '__main__':
    names = [ rndtype() + rndstring() for i in range(16) ]

    while True:
        nm = names[random.randrange(len(names))]
        val = random.gauss(5.0, 3.0)
        send_metric(nm, val, nm[0])
        rndsleep()

