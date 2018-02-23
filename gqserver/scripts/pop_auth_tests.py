import os
from Crypto.Cipher import AES
import hashlib
import time
import random


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
    # ^it took me 3 hours to realise where I went wrong.
    goal = hashlib.sha256((str(t//(12*3600))+KEY).encode()).digest()[0:16]
    print("goal: " + goal.hex())
    out = cipher.encrypt(goal)
    out = iv + out
    print("random: " + out.hex())
    out = content[0:11] + out + content[43:]
    if len(out) != len(content):
        raise Exception("Miscalculation! expecting " + str(len(content)) + " got " + str(len(out)))
    with open('../tests/auth/' + "TRUE_"+KEY+"_"+str(t),'wb') as outfile:
        outfile.write(out)
