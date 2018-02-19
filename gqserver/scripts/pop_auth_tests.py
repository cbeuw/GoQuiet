import os
from Crypto.Cipher import AES
import hashlib
import time
import random

"""
DO NOT USE pyCrypto, use pyCryptodome instead.
The AES CFB mode from pyCrypto is WRONG
I spent 3 hours trying to figure out why the test case wouldn't pass, until I read this: https://github.com/dlitz/pycrypto/issues/226
"""

TIME_HINT = 3600
KEY = "testkey"


with open('../tests/auth/.base','rb') as f:
    content = f.read()
    t = int(time.time())
    iv = random.getrandbits(128)
    iv = iv.to_bytes(16,'big')
    aes_key = hashlib.sha256(KEY.encode()).digest()
    print("aes_key: " + aes_key.hex())
    cipher = AES.new(aes_key,AES.MODE_CFB,iv,segment_size=128)
    # segment_size has to be 128 because it's default to 8 in pycryptodome, but 128 in golang crypto/aes
    goal = hashlib.sha256((str(t//TIME_HINT)+KEY).encode()).digest()[0:16]
    print("goal: " + goal.hex())
    out = cipher.encrypt(goal)
    out = iv + out
    print("random: " + out.hex())
    out = content[0:11] + out + content[43:]
    if len(out) != len(content):
        raise Exception("Miscalculation! expecting " + str(len(content)) + " got " + str(len(out)))
    with open('../tests/auth/' + "TRUE_"+KEY+"_"+str(TIME_HINT)+"_"+str(t),'wb') as outfile:
        outfile.write(out)
