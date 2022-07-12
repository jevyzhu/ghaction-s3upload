# ghaction-s3upload


## Example


```yaml

 - name: Configure AWS Credentials
   uses: aws-actions/configure-aws-credentials@v1
   with:
     aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
     aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
     aws-region: cn-north-1

 - uses: jevyzhu/ghaction-s3upload@main
   with:
     file: ${{ github.workspace }}/*.zip  // files to upload, glob supported
     max_cur: 20                          // max parallels while uploading multiple files
     s3path: mybucket/afolder  // s3 path with bucket name


```
