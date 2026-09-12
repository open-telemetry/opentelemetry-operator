<?php
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Factory\AppFactory;

require __DIR__ . '/vendor/autoload.php';

$app = AppFactory::create();

$app->get('/', function (Request $request, Response $response) {
    ob_start();
    phpinfo();
    $phpinfo = ob_get_clean();
    $response->getBody()->write($phpinfo);
    return $response->withHeader('Content-Type', 'text/html');
});

$app->run();
