## Data Chunk Flow
![未命名文件.png](https://cdn.nlark.com/yuque/0/2021/png/1732647/1622514242871-5e4e62cd-9282-4210-aba0-6784814dfb13.png#align=left&display=inline&height=685&margin=%5Bobject%20Object%5D&name=%E6%9C%AA%E5%91%BD%E5%90%8D%E6%96%87%E4%BB%B6.png&originHeight=1414&originWidth=512&size=95888&status=done&style=none&width=248)


Firstly, traverse the entire directory and calculate the size of the directory. According to the slice size, calculate the number of ouput files.


Then traverse the directory to check whether there is any unprocessed file or not.

1. If yes, go on checking whether the size of cumulative files or file slices meet the specified slice size or not.
   - If no, append file path into the small graph and repeat the process of traversing the directory to check if there is any unprocessed file.
   - If yes, continue checking if the size of files is larger than the specified slice size. If so, chunk pieces, and record the start offset and end offset of the file.
2. If no, append the files' path into the small graph, and then traverse the file directory to check if there is any unprocessed files.



At the end, process the rest of the file list, import files from one small graph and transfer them into IPLD nodes.  Then build a Merkle Tree with these nodes, write to disk and save as a CAR file.

## Restore Flow
![未命名文件 (1).png](https://cdn.nlark.com/yuque/0/2021/png/1732647/1622525954870-0663e5b3-35bc-41b9-b963-1835c7f4c245.png#align=left&display=inline&height=598&margin=%5Bobject%20Object%5D&name=%E6%9C%AA%E5%91%BD%E5%90%8D%E6%96%87%E4%BB%B6%20%281%29.png&originHeight=804&originWidth=390&size=39540&status=done&style=none&width=290)

Firstly, traverse the CAR file directory to check if there is any unprocessed file.

1. If yes, load CAR files into IPLD nodes. After processing, export files from IPLD nodes. And then traverse the CAR file directory again to check if there is any unprocessed files.
2. If no, traverse the output directory directly, search file slices according to special suffix and merge slices of the same file into one file.
