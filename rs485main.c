/*---------------------------------------------------------------------*/
/* --- 读取薄膜压力变送器的数据 ---------------------------------------*/
/* --- MCU：STC89C52RC ------------------------------------------------*/
/* --- 来源：洛迦智能开发  --------------------------------------------*/
/* --- 单片机外部晶振11.0592 , 波特率为9600---------------*/
/* --- 如果要在程序中使用此代码,请在程序中注明使用了作者的资料及程序 --*/
/*---------------------------------------------------------------------*/
 
#include <reg52.h>
#include <intrins.h>
 
#define uchar unsigned char     // 以后unsigned char就可以用uchar代替
#define uint  unsigned int      // 以后unsigned int 就可以用uint 代替
#define TRUE    1
#define FALSE   0
 
#define LCD1602_PORT P0       //液晶数据总线P0
sbit LcdEn_P    = P2^7;         // 1602液晶的EN管脚
sbit LcdRw_P    = P2^6;         // 1602液晶的RW管脚 
sbit LcdRs_P    = P2^5;         // 1602液晶的RS管脚  
 
 
sbit RE_485=P3^2;             //RS485 方向切换口
 
 
uchar  UART1_ucRec_Flag;
uchar  receBuf[9];
uchar    receTimeOut;           //接收超时
uchar  receCount;           //接收到的字节个数
uchar  fasongbiaozhi;     //查询压力值的标志位
uint   Rressure = 0;
uint   gCount=0,crcData;    // 全局计数变量
 
 
 
//字地址 0 - 255 (只取低8位)
//位地址 0 - 255 (只取低8位)
 
/* CRC 高位字节值表 */
const  uchar  code auchCRCHi[] = {
    0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0,
    0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41,
    0x00, 0xC1, 0x81, 0x40, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0,
    0x80, 0x41, 0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1, 0x81, 0x40,
    0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1,
    0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0, 0x80, 0x41,
    0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1,
    0x81, 0x40, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41,
    0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0,
    0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x00, 0xC1, 0x81, 0x40,
    0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1,
    0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1, 0x81, 0x40,
    0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0,
    0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x00, 0xC1, 0x81, 0x40,
    0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0,
    0x80, 0x41, 0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1, 0x81, 0x40,
    0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0,
    0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41,
    0x00, 0xC1, 0x81, 0x40, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0,
    0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41,
    0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0,
    0x80, 0x41, 0x00, 0xC1, 0x81, 0x40, 0x00, 0xC1, 0x81, 0x40,
    0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0, 0x80, 0x41, 0x00, 0xC1,
    0x81, 0x40, 0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41,
    0x00, 0xC1, 0x81, 0x40, 0x01, 0xC0, 0x80, 0x41, 0x01, 0xC0,
    0x80, 0x41, 0x00, 0xC1, 0x81, 0x40
} ;
 
// CRC低位字节值表
const uchar  code auchCRCLo[] = {
    0x00, 0xC0, 0xC1, 0x01, 0xC3, 0x03, 0x02, 0xC2, 0xC6, 0x06,
    0x07, 0xC7, 0x05, 0xC5, 0xC4, 0x04, 0xCC, 0x0C, 0x0D, 0xCD,
    0x0F, 0xCF, 0xCE, 0x0E, 0x0A, 0xCA, 0xCB, 0x0B, 0xC9, 0x09,
    0x08, 0xC8, 0xD8, 0x18, 0x19, 0xD9, 0x1B, 0xDB, 0xDA, 0x1A,
    0x1E, 0xDE, 0xDF, 0x1F, 0xDD, 0x1D, 0x1C, 0xDC, 0x14, 0xD4,
    0xD5, 0x15, 0xD7, 0x17, 0x16, 0xD6, 0xD2, 0x12, 0x13, 0xD3,
    0x11, 0xD1, 0xD0, 0x10, 0xF0, 0x30, 0x31, 0xF1, 0x33, 0xF3,
    0xF2, 0x32, 0x36, 0xF6, 0xF7, 0x37, 0xF5, 0x35, 0x34, 0xF4,
    0x3C, 0xFC, 0xFD, 0x3D, 0xFF, 0x3F, 0x3E, 0xFE, 0xFA, 0x3A,
    0x3B, 0xFB, 0x39, 0xF9, 0xF8, 0x38, 0x28, 0xE8, 0xE9, 0x29,
    0xEB, 0x2B, 0x2A, 0xEA, 0xEE, 0x2E, 0x2F, 0xEF, 0x2D, 0xED,
    0xEC, 0x2C, 0xE4, 0x24, 0x25, 0xE5, 0x27, 0xE7, 0xE6, 0x26,
    0x22, 0xE2, 0xE3, 0x23, 0xE1, 0x21, 0x20, 0xE0, 0xA0, 0x60,
    0x61, 0xA1, 0x63, 0xA3, 0xA2, 0x62, 0x66, 0xA6, 0xA7, 0x67,
    0xA5, 0x65, 0x64, 0xA4, 0x6C, 0xAC, 0xAD, 0x6D, 0xAF, 0x6F,
    0x6E, 0xAE, 0xAA, 0x6A, 0x6B, 0xAB, 0x69, 0xA9, 0xA8, 0x68,
    0x78, 0xB8, 0xB9, 0x79, 0xBB, 0x7B, 0x7A, 0xBA, 0xBE, 0x7E,
    0x7F, 0xBF, 0x7D, 0xBD, 0xBC, 0x7C, 0xB4, 0x74, 0x75, 0xB5,
    0x77, 0xB7, 0xB6, 0x76, 0x72, 0xB2, 0xB3, 0x73, 0xB1, 0x71,
    0x70, 0xB0, 0x50, 0x90, 0x91, 0x51, 0x93, 0x53, 0x52, 0x92,
    0x96, 0x56, 0x57, 0x97, 0x55, 0x95, 0x94, 0x54, 0x9C, 0x5C,
    0x5D, 0x9D, 0x5F, 0x9F, 0x9E, 0x5E, 0x5A, 0x9A, 0x9B, 0x5B,
    0x99, 0x59, 0x58, 0x98, 0x88, 0x48, 0x49, 0x89, 0x4B, 0x8B,
    0x8A, 0x4A, 0x4E, 0x8E, 0x8F, 0x4F, 0x8D, 0x4D, 0x4C, 0x8C,
    0x44, 0x84, 0x85, 0x45, 0x87, 0x47, 0x46, 0x86, 0x82, 0x42,
    0x43, 0x83, 0x41, 0x81, 0x80, 0x40
} ;
 
 
 
 
//=========================================================================================================================
 
 
 
 
uint  crc16(const uchar *puchMsg, uint usDataLen)
{
    uchar uchCRCHi = 0xFF ; /* 高CRC字节初始化 */
    uchar uchCRCLo = 0xFF ; /* 低CRC 字节初始化 */
    unsigned long uIndex ; /* CRC循环中的索引 */
 
    while (usDataLen--) { /* 传输消息缓冲区 */
    uIndex = uchCRCHi ^ *puchMsg++ ; /* 计算CRC */
    uchCRCHi = uchCRCLo ^ auchCRCHi[uIndex] ;
    uchCRCLo = auchCRCLo[uIndex] ;
    }
 
    return (uchCRCHi << 8 | uchCRCLo) ;
}
 
 
/*********************************************************/
// 毫秒级的延时函数，time是要延时的毫秒数
/*********************************************************/
void DelayMs(uint time)
{
    uint i,j;
    for(i=0;i<time;i++)
        for(j=0;j<112;j++);
}
 
 
 
 
/*********************************************************/
// 1602液晶写命令函数，cmd就是要写入的命令
/*********************************************************/
void LcdWriteCmd(uchar cmd)
{ 
    LcdRs_P = 0;
    LcdRw_P = 0;
    LcdEn_P = 0;
    LCD1602_PORT=cmd;
    DelayMs(2);
    LcdEn_P = 1;    
    DelayMs(2);
    LcdEn_P = 0;    
}
 
 
/*********************************************************/
// 1602液晶写数据函数，dat就是要写入的数据
/*********************************************************/
void LcdWriteData(uchar dat)
{
    LcdRs_P = 1; 
    LcdRw_P = 0;
    LcdEn_P = 0;
    LCD1602_PORT=dat;
    DelayMs(2);
    LcdEn_P = 1;    
    DelayMs(2);
    LcdEn_P = 0;
}
 
 
/*********************************************************/
// 1602液晶初始化函数
/*********************************************************/
void LcdInit()
{
    LcdWriteCmd(0x38);        // 16*2显示，5*7点阵，8位数据口
    LcdWriteCmd(0x0C);        // 开显示，不显示光标
    LcdWriteCmd(0x06);        // 地址加1，当写入数据后光标右移
    LcdWriteCmd(0x01);        // 清屏
}
 
 
/*********************************************************/
// 液晶光标定位函数
/*********************************************************/
void LcdGotoXY(uchar line,uchar column)
{
    // 第一行
    if(line==0)        
        LcdWriteCmd(0x80+column); 
     // 第二行
    if(line==1)        
        LcdWriteCmd(0x80+0x40+column); 
}
 
 
 
/*********************************************************/
//显示5位数据
/*********************************************************/
void LcdPrintNum(uint num)     
{
 
    LcdWriteData(num/10000+48); 
    LcdWriteData(num%10000/1000+48);    
    LcdWriteData(num%1000/100+48);  
    LcdWriteData(num%100/10+48);    
    LcdWriteData(num%10+48);    
}
 
 
 
/*********************************************************/
// 液晶输出字符串函数
/*********************************************************/
void LcdPrintStr(uchar *str)
{
    while(*str!='\0')
        LcdWriteData(*str++);
}
 
 
/*********************************************************/
// 液晶显示内容初始化
/*********************************************************/
void LcdShowInit()
{
    LcdGotoXY(0,0);             // 液晶光标定位到第0行第0列
    LcdPrintStr("    Rressure   ");
    LcdGotoXY(1,0);             // 液晶光标定位到第1行第0列
    LcdPrintStr("           g    ");
}
 
 
/*************************串口初始化函数**************/
void UART_Int()
{
    PCON=0x00;   //SMOD=0
    TMOD=0x21;  //设置T1为工作方式2
    TH1=0xfd;  //9600
    TL1=0xfd;
    SM0=0;
    SM1=1;//方式1,8位
    EA=1;
    ES=1;
    TR1=1;//定时器1允许
     
    TH0  = 252;     // 给定时器0的TH0装初值
    TL0  = 24;      // 给定时器0的TL0装初值 
    ET0  = 1;           // 定时器0中断使能
  TR0    = 1;           // 启动定时器0
     
    REN=1;//允许接收
}
 
 
 
/*************************串口发送一个字节函数********************/
void UART_SendChar(uchar date)
{
   ES=0;
   SBUF=date;
   while(!TI);
   TI=0;
   ES=1;
}
 
 
 
 
 
 
/*********************************************************/
// 主函数
/*********************************************************/
void main(void)
{
  LcdInit();                            // 液晶功能初始化
    LcdShowInit();                  // 液晶显示初始化
  UART_Int();             // 串口初始化,内部包含定时器0初始化
  
  while(1)
     { 
    if(fasongbiaozhi==1) //间隔100ms查询一次
            { 
                RE_485=1;              // 1代表485切换到发送模式
                DelayMs(1);            //短暂延迟
                UART_SendChar(0x01);   //发送读取数据的指令
                UART_SendChar(0X03);
                UART_SendChar(0X00);
                UART_SendChar(0X00);
                UART_SendChar(0X00);
                UART_SendChar(0X01);
                UART_SendChar(0X84);
                UART_SendChar(0X0A );
                RE_485=0;              //0代表485切换到接收模式
                fasongbiaozhi=0;
                gCount=0;           
            }
         
         
             //------------------处理串口中断函数----------------------------------        
          if(UART1_ucRec_Flag==TRUE)  //串口解析
                {
                    while(receCount <=7)
                        {
                            if(receTimeOut >= 10) //串口接收数据等待超时,跳出循环
                            break;
                        }
                         
                        if(receBuf[0]==0x01 && receBuf[1]==0x03 && receBuf[2]==0x02 ) //判断前3个字节是否正确
                            {  
                                 crcData = crc16(receBuf, 5);
                                 if (crcData == receBuf[6] + (receBuf[5]*256)) //进行CRC16效验
                                 {
                                        Rressure = receBuf[3]*256+receBuf[4] ;  // 计算压力值数据
                                        LcdGotoXY(1,5);                               // 液晶定位到第0行第5列
                                        LcdPrintNum(Rressure);                    // 显示测量结果
                                      
                                 } 
                            }
                          
                        UART1_ucRec_Flag=FALSE; 
                        receCount=0;
                 }
      }
 
}
 
 
 
 
//********************************************************
// 串口中断服务程序
//********************************************************
void UartInt(void) interrupt 4
{
 
 if(TI)                         //如果是发送中断，则不做任何处理
        {
            TI = 0;                  //清除发送中断标志位
            UART1_ucRec_Flag=FALSE;  //清忙碌标志
    }
    if(RI)                       //如果是接送中断，则进行处理
        { 
            UART1_ucRec_Flag = TRUE;
            RI = 0;                  //清除接收中断标志位        
      receTimeOut = 0;          
            receBuf[receCount] = SBUF; //将接收到的字符串存到缓存中
            receCount++;               //缓存指针向后移动
        }
 
     
} 
 
 
/*********************************************************/
// 定时器0服务程序，1毫秒
/*********************************************************/
void Timer0(void) interrupt 1
{
    TH0  = 252;                     // 给定时器0的TH0装初值
    TL0  = 24;                      // 给定时器0的TL0装初值 
     
    gCount++;                           // 每1毫秒，gCount变量加1
    receTimeOut++;
    if(receTimeOut>100) receTimeOut=100;
     
     
    if(gCount>100)               // 间隔100ms向模块发送一次指令 
    {
        gCount=0;                       // 则将gCount清零，进入新一轮的计数
        fasongbiaozhi=1;    //发送指令标志位打开
 
  }
     
     
}