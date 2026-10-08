@echo off
rem Plays the bundled story in a window.
cd /d "%~dp0"
ango.exe -window projects\intro
if errorlevel 1 pause
