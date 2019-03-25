# log4go #
---
*golang logger of log4j*

## dev list ##
---

- 2018/06/01
    > bug fix
    * output escape query string not as an string format [ as example in  TestEscapeQueryURLString ]

- 2018/03/26
    > feature
    * add async file logger
    * add json suffix for LogRecord
    * SocketLogBufferLen -> 8192
    * pack LogRecord json string instead of json.Marshal
    * add json log

    > version: 3.0.5

- 2018/03/25
    > feature
    * output colorful log to console
    * add windows new line support
    
- 2018/03/12
    > bug fix
    * delete log filter in log’s filter map in log::Close
    * change minLevel of Global from CRITICAL to DEBUG

    > version: 3.0.4

- 2017/05/02
    > bug fix
    * add sync.Once for every log Close func

- 2017/04/28
    > bug fix
    * add recover logic in file write function
    * use maxbackup instead of 999

    > feature
    * add Logger:SetAsDefaultLogger to assign a logger object to Global
    * rewrite logger:Close to guarantee it is closed once

    > version: 3.0.3

- 2017/04/24
    > improvement
    * add filename in log4go.go:intLogf & intLogc


- 2017/03/07
    > bug fix
	* log4go panic when cannot open new log file

    > version: 3.0.2

- 2017/02/09
    > bug fix
	* just closed once in log4go.go
    * add select-case in every LogWrite to avoid panic when log channel is closed.


- 2016/09/21
    > improvement
    * add Logger{FilterMap, minLevel}
    * add l4g.Close for examples/XMLConfigurationExample.go
    * delete redundant l4g.Close for examples/FileLogWriter_Manual.go
    * modify the return value to nil of log.Warn & log.Error & log.Critical
    * add Chinese remark for some key functions

