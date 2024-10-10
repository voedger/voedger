# Monitor Application

## Principles
- The main website sources kept in folder `site.main.src`
- The built version of main website kept in folder `site.main` and included into application as a static resource
- The main website is built by website developer when required 

## Requirements
The following components are required to develop and/or build the main website:
- Node.JS

## Main Website Development

```shell
# open source folder
cd ./site.main.src

# download dependencies
npm i

# start dev server
npm start
```
The development server is available by URL http://localhost:3000/static/sys/monitor/site/main/. The dev server reloads the page on changes automatically.

## Main Website Building
```shell
# open source folder
cd ./site.main.src

# download dependencies
npm i

# build
npm run build
```
The website is compiled into `site.main` directory. When app starts, the main website is available by the URL http://localhost/static/sys/monitor/site/main/